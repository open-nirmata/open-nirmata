package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"open-nirmata/db"
	"open-nirmata/db/models"
	"open-nirmata/dto"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	maxExecutionSteps       = 50                // prevent infinite loops
	maxToolIterations       = 5                 // max tool calling loops per chat stage
	defaultExecutionTimeout = 300 * time.Second // 5 minutes
)

// PromptFlowExecutorService orchestrates prompt flow execution
type PromptFlowExecutorService struct {
	chatCompletion     *ChatCompletionService
	toolExecutor       *ToolExecutorService
	knowledgeRetriever *KnowledgeRetrieverService
	db                 db.DB
}

// ExecutionContext holds the state during flow execution
type ExecutionContext struct {
	ctx            context.Context
	Execution      *models.Execution
	Flow           *models.PromptFlow
	Agent          *models.Agent
	Messages       []ChatMessage
	Variables      map[string]interface{}
	StepCount      int
	StreamCallback func(event, data interface{}) error
}

func NewPromptFlowExecutorService(database db.DB) *PromptFlowExecutorService {
	return &PromptFlowExecutorService{
		chatCompletion:     NewChatCompletionService(),
		toolExecutor:       NewToolExecutorService(),
		knowledgeRetriever: NewKnowledgeRetrieverService(),
		db:                 database,
	}
}

// ExecuteFlow executes a prompt flow from start to finish
func (s *PromptFlowExecutorService) ExecuteFlow(ctx context.Context, execution *models.Execution, agent *models.Agent, flow *models.PromptFlow, initialMessages []ChatMessage, streamCallback func(event, data interface{}) error) error {
	if execution == nil || agent == nil || flow == nil {
		return fmt.Errorf("execution, agent, and flow are required")
	}

	// Initialize execution context
	execCtx := &ExecutionContext{
		ctx:            ctx,
		Execution:      execution,
		Flow:           flow,
		Agent:          agent,
		Messages:       initialMessages,
		Variables:      make(map[string]interface{}),
		StepCount:      0,
		StreamCallback: streamCallback,
	}

	// Find entry stage
	currentStageID := flow.EntryStageID
	if currentStageID == "" && len(flow.Stages) > 0 {
		currentStageID = flow.Stages[0].Id
	}

	if currentStageID == "" {
		return fmt.Errorf("no entry stage found in flow")
	}

	// Execute stages until completion
	for {
		// Check step limit
		if execCtx.StepCount >= maxExecutionSteps {
			return fmt.Errorf("exceeded maximum execution steps (%d)", maxExecutionSteps)
		}

		// Find current stage
		stage := s.findStage(flow, currentStageID)
		if stage == nil {
			return fmt.Errorf("stage %q not found in flow", currentStageID)
		}

		if !stage.Enabled {
			// Skip disabled stages, move to first transition or end
			if len(stage.Transitions) > 0 {
				currentStageID = stage.Transitions[0].TargetStageID
				continue
			}
			break
		}

		// Execute the stage
		nextStageID, err := s.executeStage(ctx, execCtx, stage)
		if err != nil {
			// Save error step
			s.saveErrorStep(ctx, execCtx, stage, err)
			return fmt.Errorf("stage %q execution failed: %w", stage.Name, err)
		}

		execCtx.StepCount++

		// Check if execution is complete
		if nextStageID == "" {
			break
		}

		currentStageID = nextStageID
	}

	// Set final output
	if len(execCtx.Messages) > 0 {
		lastMsg := execCtx.Messages[len(execCtx.Messages)-1]
		execution.FinalOutput = lastMsg.Content
	}

	return nil
}

// executeStage executes a single stage and returns the next stage ID
func (s *PromptFlowExecutorService) executeStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage) (string, error) {
	startTime := time.Now()

	// Send stage_start event
	if execCtx.StreamCallback != nil {
		execCtx.StreamCallback("stage_start", map[string]interface{}{
			"stage_id":   stage.Id,
			"stage_name": stage.Name,
			"stage_type": stage.Type,
		})
	}

	// Create execution step
	step := &models.ExecutionStep{
		ID:            uuid.NewString(),
		StageID:       stage.Id,
		StageName:     stage.Name,
		StageType:     stage.Type,
		StartedAt:     &startTime,
		Status:        "running",
		InputMessages: s.convertToExecutionMessages(execCtx.Messages),
	}

	// Execute based on stage type
	var nextStageID string
	var err error

	switch stage.Type {
	case dto.PromptFlowStageTypeLLM:
		nextStageID, err = s.executeLLMStage(ctx, execCtx, stage, step)
	case dto.PromptFlowStageTypeResult:
		nextStageID, err = s.executeResultStage(ctx, execCtx, stage, step)
	case dto.PromptFlowStageTypeTool:
		nextStageID, err = s.executeToolStage(ctx, execCtx, stage, step)
	case dto.PromptFlowStageTypeRetrieval:
		nextStageID, err = s.executeRetrievalStage(ctx, execCtx, stage, step)
	case dto.PromptFlowStageTypeRouter:
		nextStageID, err = s.executeRouterStage(ctx, execCtx, stage, step)
	default:
		err = fmt.Errorf("unsupported stage type: %s", stage.Type)
	}

	completedTime := time.Now()
	step.CompletedAt = &completedTime

	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
	} else {
		step.Status = "completed"
		step.NextStageID = nextStageID
	}

	// Add step to execution
	execCtx.Execution.Steps = append(execCtx.Execution.Steps, *step)

	// Save to database
	s.saveExecutionStep(ctx, execCtx.Execution, step)

	// Send stage_complete event
	if execCtx.StreamCallback != nil {
		execCtx.StreamCallback("stage_complete", map[string]interface{}{
			"stage_id":      stage.Id,
			"next_stage_id": nextStageID,
			"complete":      nextStageID == "",
		})
	}

	return nextStageID, err
}

// executeLLMStage executes a chat stage with LLM
func (s *PromptFlowExecutorService) executeLLMStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, step *models.ExecutionStep) (string, error) {
	// Resolve resources (merge defaults with overrides)
	resources := s.mergeResources(execCtx.Flow.Defaults, stage.Overrides)

	// Load LLM provider
	provider, err := s.loadLLMProvider(ctx, resources.LLMProviderID)
	if err != nil {
		return "", fmt.Errorf("failed to load LLM provider: %w", err)
	}

	// Prepare tools if configured
	var tools []ChatTool
	if len(resources.ToolIDs) > 0 {
		tools, err = s.loadTools(ctx, resources.ToolIDs)
		if err != nil {
			return "", fmt.Errorf("failed to load tools: %w", err)
		}
	}

	// Build system prompt
	systemPrompt := resources.SystemPrompt
	if stage.Prompt != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n" + stage.Prompt
		} else {
			systemPrompt = stage.Prompt
		}
	}
	systemPrompt += "\n\n" + s.getResponseFormatPrompt(tools)

	// Tool calling loop
	iteration := 0
	maxIter := maxToolIterations
	if cfg := stage.Config; cfg != nil {
		if maxIterCfg, ok := cfg["max_tool_iterations"].(float64); ok && maxIterCfg > 0 {
			maxIter = int(maxIterCfg)
		}
	}

	var llmResponse *LLMResponse
	for iteration < maxIter {
		// Call LLM
		req := &ChatCompletionRequest{
			Provider:     provider,
			Model:        resources.Model,
			Messages:     execCtx.Messages,
			SystemPrompt: systemPrompt,
			Temperature:  resources.Temperature,
			// Tools:        tools,
			Stream: execCtx.StreamCallback != nil,
		}

		var response *ChatCompletionResponse
		if req.Stream {
			response, err = s.chatCompletion.ChatCompletionStream(ctx, req, 0, func(chunk *ChatCompletionStreamChunk) error {
				if chunk.Content != "" && execCtx.StreamCallback != nil {
					return execCtx.StreamCallback("llm_token", map[string]interface{}{
						"token": chunk.Content,
					})
				}
				return nil
			})
		} else {
			response, err = s.chatCompletion.ChatCompletion(ctx, req, 0)
		}

		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		// Save LLM metadata
		step.LLMCalls = append(step.LLMCalls, models.ExecutionLLMMetadata{
			ProviderID:       resources.LLMProviderID,
			Provider:         provider.Provider,
			Model:            response.Model,
			Temperature:      resources.Temperature,
			PromptTokens:     response.PromptTokens,
			CompletionTokens: response.CompletionTokens,
			TotalTokens:      response.TotalTokens,
			FinishReason:     response.FinishReason,
		})
		parsedMsg, err := s.parseLLMResponse(response.Content)
		if err != nil {
			return "", fmt.Errorf("failed to parse LLM response: %w", err)
		}
		response.ToolCalls = parsedMsg.Tools
		response.Content = parsedMsg.Response

		// Add assistant message
		assistantMsg := ChatMessage{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		}
		execCtx.Messages = append(execCtx.Messages, assistantMsg)

		// Check if there are tool calls
		if len(response.ToolCalls) == 0 {
			// No tool calls, we're done
			step.OutputMessage = s.convertToExecutionMessage(assistantMsg)
			break
		}

		// Execute tool calls
		iteration++
		hasError := false
		for _, toolCall := range response.ToolCalls {
			if execCtx.StreamCallback != nil {
				execCtx.StreamCallback("tool_call", map[string]interface{}{
					"tool_call_id": toolCall.ID,
					"tool_name":    toolCall.Function.Name,
					"arguments":    toolCall.Function.Arguments,
				})
			}

			// Find and execute tool
			toolResult, toolErr := s.executeToolCall(ctx, resources.ToolIDs, toolCall)

			execToolCall := models.ExecutionToolCall{
				ID:        toolCall.ID,
				ToolName:  toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			}

			if toolErr != nil {
				execToolCall.Error = toolErr.Error()
				hasError = true
			} else {
				execToolCall.Result = toolResult.Result
				execToolCall.ToolID = toolResult.ToolID
				execToolCall.ToolType = toolResult.ToolType
				execToolCall.LatencyMs = toolResult.LatencyMs
			}

			step.ToolCalls = append(step.ToolCalls, execToolCall)

			// Add tool result to messages
			resultContent := ""
			if toolResult != nil {
				resultContent = toolResult.Result
				if toolErr != nil {
					resultContent = fmt.Sprintf("Error: %s", toolErr.Error())
				}
			}

			toolResultMsg := ChatMessage{
				Role:       "tool",
				Content:    resultContent,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			}
			execCtx.Messages = append(execCtx.Messages, toolResultMsg)

			if execCtx.StreamCallback != nil {
				execCtx.StreamCallback("tool_result", map[string]interface{}{
					"tool_call_id": toolCall.ID,
					"tool_name":    toolCall.Function.Name,
					"result":       resultContent,
				})
			}
		}

		if !hasError {
			break
		}
	}

	// Determine next stage
	return s.selectNextStage(stage, llmResponse)
}

// executeResultStage executes a result stage with LLM
func (s *PromptFlowExecutorService) executeResultStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, step *models.ExecutionStep) (string, error) {
	// Resolve resources (merge defaults with overrides)
	resources := s.mergeResources(execCtx.Flow.Defaults, stage.Overrides)

	// Load LLM provider
	provider, err := s.loadLLMProvider(ctx, resources.LLMProviderID)
	if err != nil {
		return "", fmt.Errorf("failed to load LLM provider: %w", err)
	}

	// Build system prompt
	systemPrompt := resources.SystemPrompt
	if stage.Prompt != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n" + stage.Prompt
		} else {
			systemPrompt = stage.Prompt
		}
	}

	// Call LLM
	req := &ChatCompletionRequest{
		Provider:     provider,
		Model:        resources.Model,
		Messages:     execCtx.Messages,
		SystemPrompt: systemPrompt,
		Temperature:  resources.Temperature,
		Stream:       execCtx.StreamCallback != nil,
	}

	var response *ChatCompletionResponse
	if req.Stream {
		response, err = s.chatCompletion.ChatCompletionStream(ctx, req, 0, func(chunk *ChatCompletionStreamChunk) error {
			if chunk.Content != "" && execCtx.StreamCallback != nil {
				return execCtx.StreamCallback("llm_token", map[string]interface{}{
					"token": chunk.Content,
				})
			}
			return nil
		})
	} else {
		response, err = s.chatCompletion.ChatCompletion(ctx, req, 0)
	}

	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Save LLM metadata
	step.LLMCalls = append(step.LLMCalls, models.ExecutionLLMMetadata{
		ProviderID:       resources.LLMProviderID,
		Provider:         provider.Provider,
		Model:            response.Model,
		Temperature:      resources.Temperature,
		PromptTokens:     response.PromptTokens,
		CompletionTokens: response.CompletionTokens,
		TotalTokens:      response.TotalTokens,
		FinishReason:     response.FinishReason,
	})

	// Add assistant message
	assistantMsg := ChatMessage{
		Role:      "assistant",
		Content:   response.Content,
		ToolCalls: response.ToolCalls,
	}
	execCtx.Messages = append(execCtx.Messages, assistantMsg)

	// Determine next stage
	return s.selectNextStage(stage, nil)
}

// executeToolStage executes a tool stage
func (s *PromptFlowExecutorService) executeToolStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, step *models.ExecutionStep) (string, error) {
	// Tool stage executes specific tools defined in config
	// Config format: { "tool_calls": [{ "tool_name": "name", "arguments": {...} }] }

	if stage.Config == nil {
		return "", fmt.Errorf("tool stage requires config with tool_calls")
	}

	toolCallsRaw, ok := stage.Config["tool_calls"]
	if !ok {
		return "", fmt.Errorf("tool stage config must include 'tool_calls' array")
	}

	toolCallsList, ok := toolCallsRaw.([]interface{})
	if !ok {
		return "", fmt.Errorf("tool_calls must be an array")
	}

	if len(toolCallsList) == 0 {
		return s.selectNextStage(stage, nil)
	}

	// Get resources to determine which tools are available
	resources := s.mergeResources(execCtx.Flow.Defaults, stage.Overrides)
	if len(resources.ToolIDs) == 0 {
		return "", fmt.Errorf("no tools configured in resources")
	}

	// Execute each tool call
	var resultMessages []string

	for i, toolCallRaw := range toolCallsList {
		toolCallMap, ok := toolCallRaw.(map[string]interface{})
		if !ok {
			step.Error = fmt.Sprintf("tool_call[%d] must be an object", i)
			continue
		}

		toolName, ok := toolCallMap["tool_name"].(string)
		if !ok || toolName == "" {
			step.Error = fmt.Sprintf("tool_call[%d] missing tool_name", i)
			continue
		}

		arguments, ok := toolCallMap["arguments"].(map[string]interface{})
		if !ok {
			arguments = make(map[string]interface{})
		}

		// Stream callback for tool execution
		if execCtx.StreamCallback != nil {
			execCtx.StreamCallback("tool_call", map[string]interface{}{
				"tool_call_id": fmt.Sprintf("tool-stage-%d", i),
				"tool_name":    toolName,
				"arguments":    arguments,
			})
		}

		// Find and execute the tool
		toolID := uuid.NewString()
		toolCall := ChatToolCall{
			ID:   toolID,
			Type: "function",
			Function: ChatToolCallFunction{
				Name:      toolName,
				Arguments: arguments,
			},
		}

		startTime := time.Now()
		toolResult, toolErr := s.executeToolCall(ctx, resources.ToolIDs, toolCall)
		latencyMs := time.Since(startTime).Milliseconds()

		execToolCall := models.ExecutionToolCall{
			ID:        toolID,
			ToolName:  toolName,
			Arguments: arguments,
			StartedAt: &startTime,
		}
		completedTime := time.Now()
		execToolCall.CompletedAt = &completedTime
		execToolCall.LatencyMs = latencyMs

		if toolErr != nil {
			execToolCall.Error = toolErr.Error()
			resultMessages = append(resultMessages, fmt.Sprintf("%s: Error - %s", toolName, toolErr.Error()))
		} else {
			execToolCall.Result = toolResult.Result
			execToolCall.ToolID = toolResult.ToolID
			execToolCall.ToolType = toolResult.ToolType
			resultMessages = append(resultMessages, fmt.Sprintf("%s: %s", toolName, toolResult.Result))
		}

		step.ToolCalls = append(step.ToolCalls, execToolCall)

		// Stream callback for tool result
		if execCtx.StreamCallback != nil {
			execCtx.StreamCallback("tool_result", map[string]interface{}{
				"tool_call_id": toolID,
				"tool_name":    toolName,
				"result":       execToolCall.Result,
				"error":        execToolCall.Error,
			})
		}
	}

	// Add results to conversation context if configured
	addToContext := true
	if addToCtxRaw, ok := stage.Config["add_to_context"]; ok {
		if addToCtxBool, ok := addToCtxRaw.(bool); ok {
			addToContext = addToCtxBool
		}
	}

	if addToContext && len(resultMessages) > 0 {
		// Combine all tool results into a single message
		combinedResult := strings.Join(resultMessages, "\n")

		toolResultMsg := ChatMessage{
			Role:    "tool",
			Content: combinedResult,
			Name:    "tool_stage",
		}
		execCtx.Messages = append(execCtx.Messages, toolResultMsg)
		step.OutputMessage = s.convertToExecutionMessage(toolResultMsg)
	}

	return s.selectNextStage(stage, nil)
}

// executeRetrievalStage executes a retrieval stage
func (s *PromptFlowExecutorService) executeRetrievalStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, step *models.ExecutionStep) (string, error) {
	resources := s.mergeResources(execCtx.Flow.Defaults, stage.Overrides)

	if len(resources.KnowledgebaseIDs) == 0 {
		return s.selectNextStage(stage, nil)
	}

	// Load knowledgebases
	knowledgebases, err := s.loadKnowledgebases(ctx, resources.KnowledgebaseIDs)
	if err != nil {
		return "", fmt.Errorf("failed to load knowledgebases: %w", err)
	}

	// Get query from stage prompt or last user message
	query := stage.Prompt
	if query == "" {
		// Use last user message as query
		for i := len(execCtx.Messages) - 1; i >= 0; i-- {
			if execCtx.Messages[i].Role == "user" {
				query = execCtx.Messages[i].Content
				break
			}
		}
	}

	// Retrieve context
	topK := 5
	if cfg := stage.Config; cfg != nil {
		if topKCfg, ok := cfg["top_k"].(float64); ok {
			topK = int(topKCfg)
		}
	}

	results, err := s.knowledgeRetriever.RetrieveContext(ctx, &RetrievalRequest{
		Knowledgebases: knowledgebases,
		Query:          query,
		TopK:           topK,
	}, 0)

	if err != nil {
		return "", fmt.Errorf("knowledge retrieval failed: %w", err)
	}

	// Format and add to context
	if len(results) > 0 {
		contextStr := s.knowledgeRetriever.FormatRetrievedContext(results)

		// Store in step
		for _, r := range results {
			step.RetrievedContext = append(step.RetrievedContext, r.Content)
		}

		// Add as system message or user context
		contextMsg := ChatMessage{
			Role:    "user",
			Content: contextStr,
		}
		execCtx.Messages = append(execCtx.Messages, contextMsg)
		step.OutputMessage = s.convertToExecutionMessage(contextMsg)
	}

	return s.selectNextStage(stage, nil)
}

// executeRouterStage executes a router stage using LLM
func (s *PromptFlowExecutorService) executeRouterStage(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, step *models.ExecutionStep) (string, error) {
	if len(stage.Transitions) == 0 {
		return "", fmt.Errorf("router stage must have at least one transition")
	}

	// Use LLM to classify and select next stage
	resources := s.mergeResources(execCtx.Flow.Defaults, stage.Overrides)

	provider, err := s.loadLLMProvider(ctx, resources.LLMProviderID)
	if err != nil {
		// Fallback to first transition if LLM provider not available
		step.TransitionReason = "fallback: LLM provider not available"
		return stage.Transitions[0].TargetStageID, nil
	}

	// Build classification prompt
	prompt := s.buildRouterPrompt(stage, execCtx.Messages)

	// Call LLM for classification
	req := &ChatCompletionRequest{
		Provider: provider,
		Model:    resources.Model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: resources.Temperature,
	}

	response, err := s.chatCompletion.ChatCompletion(ctx, req, 0)
	if err != nil {
		// Fallback to first transition
		step.TransitionReason = "fallback: LLM call failed"
		return stage.Transitions[0].TargetStageID, nil
	}

	// Parse LLM response to extract target stage ID
	selectedStageID := s.parseRouterResponse(response.Content, stage.Transitions)
	if selectedStageID == "" {
		// Fallback to first transition
		step.TransitionReason = "fallback: could not parse LLM response"
		return stage.Transitions[0].TargetStageID, nil
	}

	step.TransitionReason = fmt.Sprintf("LLM selected: %s", selectedStageID)
	return selectedStageID, nil
}

// Helper methods

func (s *PromptFlowExecutorService) findStage(flow *models.PromptFlow, stageID string) *models.PromptFlowStage {
	for i := range flow.Stages {
		if flow.Stages[i].Id == stageID {
			return &flow.Stages[i]
		}
	}
	return nil
}

func (s *PromptFlowExecutorService) mergeResources(defaults, overrides *models.PromptFlowResources) *models.PromptFlowResources {
	result := &models.PromptFlowResources{}

	if defaults != nil {
		*result = *defaults
	}

	if overrides != nil {
		if overrides.LLMProviderID != "" {
			result.LLMProviderID = overrides.LLMProviderID
		}
		if overrides.Model != "" {
			result.Model = overrides.Model
		}
		if overrides.SystemPrompt != "" {
			result.SystemPrompt = overrides.SystemPrompt
		}
		if overrides.Temperature != nil {
			result.Temperature = overrides.Temperature
		}
		if len(overrides.ToolIDs) > 0 {
			result.ToolIDs = overrides.ToolIDs
		}
		if len(overrides.KnowledgebaseIDs) > 0 {
			result.KnowledgebaseIDs = overrides.KnowledgebaseIDs
		}
		if overrides.Metadata != nil {
			result.Metadata = overrides.Metadata
		}
	}

	return result
}

func (s *PromptFlowExecutorService) selectNextStage(stage *models.PromptFlowStage, res *LLMResponse) (string, error) {
	if stage.Type == dto.PromptFlowStageTypeResult {
		return "", nil
	}
	if stage.Type == dto.PromptFlowStageTypeTool || stage.Type == dto.PromptFlowStageTypeRetrieval || stage.Type == dto.PromptFlowStageTypeLLM {
		return stage.OnSuccess, nil
	}
	if len(stage.Transitions) == 0 {
		return "", nil // No more stages
	}

	// ideally we should not readch here as router is handled separately, but just in case, we can evaluate conditions here

	// For now, just take the first transition
	// TODO: Implement condition evaluation for non-router stages
	return stage.Transitions[0].TargetStageID, nil
}

func (s *PromptFlowExecutorService) buildRouterPrompt(stage *models.PromptFlowStage, messages []ChatMessage) string {
	prompt := "Based on the conversation context, select the most appropriate next step.\n\n"

	// Add conversation context (last few messages)
	prompt += "Recent conversation:\n"
	startIdx := len(messages) - 5
	if startIdx < 0 {
		startIdx = 0
	}
	for _, msg := range messages[startIdx:] {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	prompt += "\nAvailable options:\n"
	for _, trans := range stage.Transitions {
		label := trans.Label
		if label == "" {
			label = trans.TargetStageID
		}
		prompt += fmt.Sprintf("%s. %s", label, trans.TargetStageID)
		if trans.Condition != "" {
			prompt += fmt.Sprintf(" (when: %s)", trans.Condition)
		}
		prompt += "\n"
	}

	prompt += "\nRespond with ONLY the target_stage_id of your choice, nothing else."
	return prompt
}

func (s *PromptFlowExecutorService) parseRouterResponse(response string, transitions []models.PromptFlowTransition) string {
	response = strings.TrimSpace(strings.ToLower(response))

	// Try to find exact match
	for _, trans := range transitions {
		if strings.Contains(response, strings.ToLower(trans.TargetStageID)) {
			return trans.TargetStageID
		}
	}

	return ""
}

func (s *PromptFlowExecutorService) convertToExecutionMessages(messages []ChatMessage) []models.ExecutionMessage {
	result := make([]models.ExecutionMessage, 0, len(messages))
	for _, msg := range messages {
		execMsg := models.ExecutionMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}

		for _, tc := range msg.ToolCalls {
			execMsg.ToolCalls = append(execMsg.ToolCalls, models.ExecutionToolCall{
				ID:        tc.ID,
				ToolName:  tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		result = append(result, execMsg)
	}
	return result
}

func (s *PromptFlowExecutorService) convertToExecutionMessage(msg ChatMessage) *models.ExecutionMessage {
	execMsg := &models.ExecutionMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
	}

	for _, tc := range msg.ToolCalls {
		execMsg.ToolCalls = append(execMsg.ToolCalls, models.ExecutionToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return execMsg
}

func (s *PromptFlowExecutorService) saveExecutionStep(ctx context.Context, execution *models.Execution, step *models.ExecutionStep) error {
	if s.db == nil {
		return nil
	}

	model := models.GetExecutionModel()
	filter := bson.M{model.IdKey: execution.ID}
	update := bson.M{
		"$push": bson.M{model.StepsKey: step},
		"$set":  bson.M{model.UpdatedAtKey: time.Now()},
	}

	_, err := s.db.UpdateOne(ctx, model, filter, update)
	return err
}

func (s *PromptFlowExecutorService) saveErrorStep(ctx context.Context, execCtx *ExecutionContext, stage *models.PromptFlowStage, err error) {
	now := time.Now()
	step := &models.ExecutionStep{
		ID:          uuid.NewString(),
		StageID:     stage.Id,
		StageName:   stage.Name,
		StageType:   stage.Type,
		StartedAt:   &now,
		CompletedAt: &now,
		Status:      "failed",
		Error:       err.Error(),
	}

	execCtx.Execution.Steps = append(execCtx.Execution.Steps, *step)
	s.saveExecutionStep(ctx, execCtx.Execution, step)
}

// Database loaders

func (s *PromptFlowExecutorService) loadLLMProvider(ctx context.Context, providerID string) (*models.LLMProvider, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	model := models.GetLLMProviderModel()
	filter := bson.M{model.IdKey: providerID}

	var provider models.LLMProvider
	if err := s.db.FindOne(ctx, model, filter).Decode(&provider); err != nil {
		return nil, err
	}

	return &provider, nil
}

func (s *PromptFlowExecutorService) loadTools(ctx context.Context, toolIDs []string) ([]ChatTool, error) {
	if s.db == nil || len(toolIDs) == 0 {
		return []ChatTool{}, nil
	}

	model := models.GetToolModel()
	filter := bson.M{model.IdKey: bson.M{"$in": toolIDs}}

	cursor, err := s.db.Find(ctx, model, filter, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tools []models.Tool
	if err := cursor.All(ctx, &tools); err != nil {
		return nil, err
	}
	// if the tool is http we need to convert it to a chat tool with function calling format

	chatTools := make([]ChatTool, 0, len(tools))
	for _, tool := range tools {
		switch strings.ToLower(strings.TrimSpace(tool.Type)) {
		case string(dto.ToolTypeHTTP):
			chatTools = append(chatTools, buildHTTPChatTool(tool))
		default:
			chatTools = append(chatTools, buildChatToolsForTool(tool)...)
		}
	}

	return chatTools, nil
}

func buildChatToolsForTool(tool models.Tool) []ChatTool {
	if len(tool.Tools) > 0 {
		chatTools := make([]ChatTool, 0, len(tool.Tools))
		for i := range tool.Tools {
			discoveredTool := tool.Tools[i]
			name := strings.TrimSpace(discoveredTool.Name)
			if name == "" {
				continue
			}

			description := strings.TrimSpace(discoveredTool.Description)
			if description == "" {
				description = tool.Description
			}

			chatTools = append(chatTools, ChatTool{
				Type: "function",
				Function: ChatToolFunction{
					Name:        name,
					Description: description,
					Parameters:  buildToolParametersSchema(tool, &discoveredTool),
				},
			})
		}
		if len(chatTools) > 0 {
			return chatTools
		}
	}

	return []ChatTool{{
		Type: "function",
		Function: ChatToolFunction{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  buildToolParametersSchema(tool, nil),
		},
	}}
}

func buildHTTPChatTool(tool models.Tool) ChatTool {
	description := strings.TrimSpace(tool.Description)
	if description == "" && tool.Config != nil {
		method := strings.ToUpper(strings.TrimSpace(tool.Config.Method))
		if method == "" {
			method = "GET"
		}
		url := strings.TrimSpace(tool.Config.URL)
		if url != "" {
			description = fmt.Sprintf("Invoke HTTP endpoint %s %s", method, url)
		}
	}

	return ChatTool{
		Type: "function",
		Function: ChatToolFunction{
			Name:        strings.TrimSpace(tool.Name),
			Description: description,
			Parameters:  buildHTTPToolParametersSchema(tool.Config),
		},
	}
}

func buildToolParametersSchema(tool models.Tool, discoveredTool *dto.MCPDiscoveredTool) map[string]interface{} {
	defaultSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	if discoveredTool != nil && len(discoveredTool.InputSchema) > 0 {
		return normalizeToolParametersSchema(discoveredTool.InputSchema)
	}
	if tool.Config != nil && len(tool.Config.InputSchema) > 0 {
		return normalizeToolParametersSchema(tool.Config.InputSchema)
	}
	if strings.EqualFold(strings.TrimSpace(tool.Type), string(dto.ToolTypeHTTP)) {
		return buildHTTPToolParametersSchema(tool.Config)
	}
	if len(tool.Tools) == 1 && len(tool.Tools[0].InputSchema) > 0 {
		return normalizeToolParametersSchema(tool.Tools[0].InputSchema)
	}

	return defaultSchema
}

func buildHTTPToolParametersSchema(config *dto.ToolConfig) map[string]interface{} {
	defaultSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
	if config == nil {
		return defaultSchema
	}
	if len(config.InputSchema) > 0 {
		return normalizeToolParametersSchema(config.InputSchema)
	}

	properties := map[string]interface{}{}
	requiredSet := map[string]struct{}{}

	for _, value := range []string{config.URL, config.PayloadTemplate} {
		for _, variable := range extractTemplateVariables(value) {
			properties[variable] = map[string]interface{}{"type": "string"}
			requiredSet[variable] = struct{}{}
		}
	}

	for _, value := range config.QueryParams {
		for _, variable := range extractTemplateVariables(value) {
			properties[variable] = map[string]interface{}{"type": "string"}
			requiredSet[variable] = struct{}{}
		}
	}

	if len(properties) == 0 {
		return defaultSchema
	}

	required := make([]string, 0, len(requiredSet))
	for name := range requiredSet {
		required = append(required, name)
	}
	sort.Strings(required)

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		requiredValues := make([]interface{}, 0, len(required))
		for _, name := range required {
			requiredValues = append(requiredValues, name)
		}
		schema["required"] = requiredValues
	}

	return schema
}

var templateVariablePattern = regexp.MustCompile(`\{\{\s*\.?([a-zA-Z_][a-zA-Z0-9_\.]*)[^}]*\}\}`)

func extractTemplateVariables(templateStr string) []string {
	matches := templateVariablePattern.FindAllStringSubmatch(templateStr, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	variables := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		name := strings.TrimSpace(match[1])
		if idx := strings.Index(name, "."); idx >= 0 {
			name = name[:idx]
		}
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}

		seen[name] = struct{}{}
		variables = append(variables, name)
	}

	return variables
}

func normalizeToolParametersSchema(inputSchema map[string]interface{}) map[string]interface{} {
	parameters := make(map[string]interface{}, len(inputSchema)+1)
	for key, value := range inputSchema {
		parameters[key] = value
	}

	if _, ok := parameters["type"]; !ok {
		parameters["type"] = "object"
	}
	if paramType, ok := parameters["type"].(string); ok && strings.EqualFold(paramType, "object") {
		if _, hasProperties := parameters["properties"]; !hasProperties {
			parameters["properties"] = map[string]interface{}{}
		}
	}

	return parameters
}

func (s *PromptFlowExecutorService) loadKnowledgebases(ctx context.Context, kbIDs []string) ([]*models.Knowledgebase, error) {
	if s.db == nil || len(kbIDs) == 0 {
		return []*models.Knowledgebase{}, nil
	}

	model := models.GetKnowledgebaseModel()
	filter := bson.M{model.IdKey: bson.M{"$in": kbIDs}}

	cursor, err := s.db.Find(ctx, model, filter, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var kbs []models.Knowledgebase
	if err := cursor.All(ctx, &kbs); err != nil {
		return nil, err
	}

	result := make([]*models.Knowledgebase, len(kbs))
	for i := range kbs {
		result[i] = &kbs[i]
	}

	return result, nil
}

func (s *PromptFlowExecutorService) executeToolCall(ctx context.Context, toolIDs []string, toolCall ChatToolCall) (*ToolExecutionResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	model := models.GetToolModel()
	filter := bson.M{model.IdKey: bson.M{"$in": toolIDs}}

	cursor, err := s.db.Find(ctx, model, filter, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tools []models.Tool
	if err := cursor.All(ctx, &tools); err != nil {
		return nil, err
	}

	tool, err := findToolForExecution(tools, toolCall.Function.Name)
	if err != nil {
		return nil, err
	}

	execReq := &ToolExecutionRequest{
		Tool:      tool,
		ToolName:  toolCall.Function.Name,
		Arguments: toolCall.Function.Arguments,
	}

	return s.toolExecutor.ExecuteTool(ctx, execReq, 0)
}

func findToolForExecution(tools []models.Tool, toolName string) (*models.Tool, error) {
	trimmedName := strings.TrimSpace(toolName)
	for i := range tools {
		if strings.EqualFold(strings.TrimSpace(tools[i].Name), trimmedName) {
			return &tools[i], nil
		}
	}

	for i := range tools {
		for _, discoveredTool := range tools[i].Tools {
			if strings.EqualFold(strings.TrimSpace(discoveredTool.Name), trimmedName) {
				return &tools[i], nil
			}
		}
	}

	return nil, fmt.Errorf("tool %q not found", toolName)
}

func (s *PromptFlowExecutorService) getTransitionPrompt(p *models.PromptFlowStage) string {
	if len(p.Transitions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Determine the next step to take:\n\n")
	for _, t := range p.Transitions {
		if t.Condition == "" {
			sb.WriteString(fmt.Sprintf("- If %s, then %s\n", t.Label, t.TargetStageID))
		} else {
			sb.WriteString(fmt.Sprintf("- If %s with %s, then %s\n", t.Condition, t.Label, t.TargetStageID))
		}
	}
	return sb.String()
}

func (s *PromptFlowExecutorService) getResponseFormatPrompt(tools []ChatTool) string {
	toolDefinitions := "[]"
	if len(tools) > 0 {
		if data, err := json.MarshalIndent(tools, "", "  "); err == nil {
			toolDefinitions = string(data)
		}
	}

	return fmt.Sprintf(`You must respond with ONLY valid JSON matching this exact structure (no markdown fences and no extra prose):
`+"```json"+`
{
  "tools": [
    {
      "id": "tool-call-1",
      "type": "function",
      "function": {
        "name": "tool_name",
        "arguments": {
          "key": "value"
        }
      }
    }
  ],
  "tool_calls": [
    {
      "tool_name": "tool_name",
      "arguments": {
        "key": "value"
      }
    }
  ],
  "response": "assistant response text",
  "variables": {
    "key": "value"
  }
}
`+"```"+`
Available tool definitions:
%s

Rules:
- Use only tools from the definitions above.
- "tools" must always be an array; use [] if no tool is needed.
- "tool_calls" must always be an array in the simplified format expected by the tool stage.
- When you call a tool, keep "tools" and "tool_calls" aligned so they describe the same action.
- Each item in "tools" must include "id", "type":"function", and "function".
- "function.arguments" and "tool_calls[].arguments" must be JSON objects, not stringified JSON blobs.
- Put the user-facing reply in "response". It may be an empty string when you need tools first.
- "variables" is optional; only include it when you want to store values for later stages.
- Return JSON only so the tool-calling stage can consume it directly.`, toolDefinitions)
}

func (s *PromptFlowExecutorService) parseLLMResponse(response string) (*LLMResponse, error) {
	// For simplicity, assume response is in format:
	// { "response": "text response", "next_stage_id" : "id of next stage or empty", "variables": {"key": "value"} }
	// there is a chance the entire thing might be wrapped in markdown, so we should try to extract the JSON from markdown if needed

	// Try to extract JSON from markdown
	jsonStr := response
	if strings.HasPrefix(response, "```") && strings.HasSuffix(response, "```") {
		// take the text between ```json and ```. Ignore any text outside of that
		// response format might be Text response ..... ```json {json} ```
		startIdx := strings.Index(response, "```json")
		if startIdx == -1 {
			startIdx = strings.Index(response, "```")
			if startIdx == -1 {
				return nil, fmt.Errorf("response is wrapped in markdown but no code block found")
			}
		} else {
			startIdx += len("```json")
		}

		endIdx := strings.LastIndex(response, "```")
		if endIdx == -1 || endIdx <= startIdx {
			return nil, fmt.Errorf("response is wrapped in markdown but no closing code block found")
		}

		jsonStr = response[startIdx:endIdx]
		jsonStr = strings.TrimSpace(jsonStr)
	}

	var parsed struct {
		Response  string         `json:"response"`
		Tools     []ChatToolCall `json:"tools"`
		ToolCalls []struct {
			ToolName  string                 `json:"tool_name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"tool_calls"`
		Variables map[string]interface{} `json:"variables"`
	}

	err := json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(parsed.Tools) == 0 && len(parsed.ToolCalls) > 0 {
		parsed.Tools = make([]ChatToolCall, 0, len(parsed.ToolCalls))
		for i, toolCall := range parsed.ToolCalls {
			name := strings.TrimSpace(toolCall.ToolName)
			if name == "" {
				continue
			}

			parsed.Tools = append(parsed.Tools, ChatToolCall{
				ID:   fmt.Sprintf("tool-call-%d", i+1),
				Type: "function",
				Function: ChatToolCallFunction{
					Name:      name,
					Arguments: toolCall.Arguments,
				},
			})
		}
	}

	for i := range parsed.Tools {
		if strings.TrimSpace(parsed.Tools[i].ID) == "" {
			parsed.Tools[i].ID = fmt.Sprintf("tool-call-%d", i+1)
		}
		if strings.TrimSpace(parsed.Tools[i].Type) == "" {
			parsed.Tools[i].Type = "function"
		}
	}

	return &LLMResponse{
		Response:  parsed.Response,
		Tools:     parsed.Tools,
		Variables: parsed.Variables,
	}, nil
}

type LLMResponse struct {
	Response  string
	Tools     []ChatToolCall
	Variables map[string]interface{}
}
