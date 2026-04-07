package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"open-nirmata/db"
	"open-nirmata/db/models"
	"open-nirmata/dto"
	"open-nirmata/providers"
	"open-nirmata/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExecuteAgent executes an agent with a user message
func ExecuteAgent(c *fiber.Ctx) error {
	agentID := strings.TrimSpace(c.Params("id"))
	if agentID == "" {
		return badRequest(c, "agent id is required")
	}

	var req dto.ExecuteAgentRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	// Validate input
	if req.Message == "" && len(req.Messages) == 0 {
		return badRequest(c, "either message or messages is required")
	}

	// Check for async mode
	async := strings.ToLower(c.Query("async")) == "true"
	stream := req.Stream && !async // streaming only works in sync mode

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	// Load agent
	agent, err := loadAgentByRecordID(c, agentID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "agent not found")
		}
		return internalError(c, "failed to load agent")
	}

	if !agent.Enabled {
		return badRequest(c, "agent is disabled")
	}

	// Load prompt flow
	flow, err := loadPromptFlow(c, serviceProvider.D, agent.PromptFlowID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return badRequest(c, "prompt flow not found")
		}
		return internalError(c, "failed to load prompt flow")
	}

	if !flow.Enabled {
		return badRequest(c, "prompt flow is disabled")
	}

	// Create execution record
	now := time.Now().UTC()
	execution := &models.Execution{
		ID:           uuid.NewString(),
		AgentID:      agent.Id,
		PromptFlowID: flow.Id,
		Status:       models.ExecutionStatusRunning,
		Input:        buildExecutionInput(req),
		Steps:        []models.ExecutionStep{},
		CreatedAt:    &now,
		UpdatedAt:    &now,
		Metadata:     req.Metadata,
	}

	// Save initial execution record
	execModel := models.GetExecutionModel()
	if _, err := serviceProvider.D.InsertOne(c.Context(), execModel, execution); err != nil {
		return internalError(c, "failed to create execution")
	}

	// Build initial messages
	initialMessages := buildInitialMessages(req, flow.IncludeConversationHistory)

	if async {
		// Async mode: return job ID immediately and run in background
		go executeFlowAsync(execution, &agent, flow, initialMessages, serviceProvider.D)

		return c.JSON(dto.ExecuteAgentResponse{
			Success: true,
			JobID:   execution.ID,
			Message: "execution started",
		})
	}

	if stream {
		// Streaming mode: use SSE
		return executeFlowStreaming(c, execution, &agent, flow, initialMessages, serviceProvider.D)
	}

	// Synchronous mode: execute and return result
	return executeFlowSync(c, execution, &agent, flow, initialMessages, serviceProvider.D)
}

// GetExecution retrieves an execution by ID
func GetExecution(c *fiber.Ctx) error {
	executionID := strings.TrimSpace(c.Params("id"))
	if executionID == "" {
		return badRequest(c, "execution id is required")
	}

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	execModel := models.GetExecutionModel()
	filter := bson.M{execModel.IdKey: executionID}

	var execution models.Execution
	if err := serviceProvider.D.FindOne(c.Context(), execModel, filter).Decode(&execution); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "execution not found")
		}
		return internalError(c, "failed to load execution")
	}

	execItem := toExecutionItem(execution)
	return c.JSON(dto.GetExecutionResponse{
		Success: true,
		Data:    &execItem,
	})
}

// ListExecutions retrieves recent executions, optionally filtered by agent and status.
func ListExecutions(c *fiber.Ctx) error {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	execModel := models.GetExecutionModel()
	filter := bson.M{}

	if agentID := strings.TrimSpace(c.Query("agent_id")); agentID != "" {
		filter[execModel.AgentIDKey] = agentID
	}

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter[execModel.StatusKey] = status
	}

	limit := int64(20)
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed < 1 {
			return badRequest(c, "limit must be a positive integer")
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: execModel.CreatedAtKey, Value: -1}}).
		SetLimit(limit)

	cursor, err := serviceProvider.D.Find(c.Context(), execModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list executions")
	}
	defer cursor.Close(c.Context())

	executions := make([]dto.ExecutionItem, 0)
	for cursor.Next(c.Context()) {
		var execution models.Execution
		if err := cursor.Decode(&execution); err != nil {
			return internalError(c, "failed to decode execution")
		}
		executions = append(executions, toExecutionItem(execution))
	}

	if err := cursor.Err(); err != nil {
		return internalError(c, "failed to read executions")
	}

	count, err := serviceProvider.D.CountDocuments(c.Context(), execModel, filter)
	if err != nil {
		return internalError(c, "failed to count executions")
	}

	return c.JSON(dto.ListExecutionsResponse{
		Success: true,
		Data:    executions,
		Count:   int(count),
	})
}

// Helper functions

func buildExecutionInput(req dto.ExecuteAgentRequest) map[string]interface{} {
	input := make(map[string]interface{})

	if req.Message != "" {
		input["message"] = req.Message
	}

	if len(req.Messages) > 0 {
		input["messages"] = req.Messages
	}

	return input
}

func buildInitialMessages(req dto.ExecuteAgentRequest, includeConversationHistory *bool) []services.ChatMessage {
	messages := make([]services.ChatMessage, 0)
	if req.Message != "" {
		// Single message mode
		messages = append(messages, services.ChatMessage{
			Role:    "user",
			Content: req.Message,
		})
	}

	if includeConversationHistory == nil || !*includeConversationHistory {
		return messages
	}

	for _, msg := range req.Messages {
		messages = append(messages, toChatMessage(msg))
	}

	return messages
}

func toChatMessage(msg dto.ExecutionMessageItem) services.ChatMessage {
	chatMsg := services.ChatMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
	}

	for _, tc := range msg.ToolCalls {
		chatMsg.ToolCalls = append(chatMsg.ToolCalls, services.ChatToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: services.ChatToolCallFunction{
				Name:      tc.ToolName,
				Arguments: tc.Arguments,
			},
		})
	}

	return chatMsg
}

func executeFlowSync(c *fiber.Ctx, execution *models.Execution, agent *models.Agent, flow *models.PromptFlow, initialMessages []services.ChatMessage, database db.DB) error {
	startTime := time.Now()

	executor := services.NewPromptFlowExecutorService(database)

	// Execute flow
	if err := executor.ExecuteFlow(c.Context(), execution, agent, flow, initialMessages, nil); err != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = err.Error()
		completedTime := time.Now()
		execution.CompletedAt = &completedTime
		execution.UpdatedAt = &completedTime
		execution.TotalLatencyMs = completedTime.Sub(startTime).Milliseconds()

		// Update execution in database
		execModel := models.GetExecutionModel()
		filter := bson.M{execModel.IdKey: execution.ID}
		update := bson.M{"$set": execution}
		database.UpdateOne(c.Context(), execModel, filter, update)

		return internalError(c, fmt.Sprintf("execution failed: %s", err.Error()))
	}

	// Update execution status
	execution.Status = models.ExecutionStatusCompleted
	completedTime := time.Now()
	execution.CompletedAt = &completedTime
	execution.UpdatedAt = &completedTime
	execution.TotalLatencyMs = completedTime.Sub(startTime).Milliseconds()

	execModel := models.GetExecutionModel()
	filter := bson.M{execModel.IdKey: execution.ID}
	update := bson.M{"$set": execution}
	database.UpdateOne(c.Context(), execModel, filter, update)

	execItem := toExecutionItem(*execution)
	return c.JSON(dto.ExecuteAgentResponse{
		Success: true,
		Data:    &execItem,
		Message: "execution completed",
	})
}

func executeFlowStreaming(c *fiber.Ctx, execution *models.Execution, agent *models.Agent, flow *models.PromptFlow, initialMessages []services.ChatMessage, database db.DB) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		startTime := time.Now()

		// Get executor service
		executor := services.NewPromptFlowExecutorService(database)

		// Use background context since we're in a separate goroutine
		ctx := context.Background()

		// Stream callback
		streamCallback := func(event interface{}, data interface{}) error {
			eventName := ""
			if e, ok := event.(string); ok {
				eventName = e
			}
			sseEvent := dto.SSEEvent{
				Event: eventName,
				Data:  data,
			}

			eventJSON, _ := json.Marshal(sseEvent)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, eventJSON)
			w.Flush()
			return nil
		}

		// Execute flow
		err := executor.ExecuteFlow(ctx, execution, agent, flow, initialMessages, streamCallback)

		// Send completion or error event
		if err != nil {
			execution.Status = models.ExecutionStatusFailed
			execution.Error = err.Error()

			errorEvent := dto.SSEEvent{
				Event: dto.SSEEventError,
				Data: dto.SSEErrorData{
					Error: err.Error(),
				},
			}
			eventJSON, _ := json.Marshal(errorEvent)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", dto.SSEEventError, eventJSON)
		} else {
			execution.Status = models.ExecutionStatusCompleted

			completeEvent := dto.SSEEvent{
				Event: dto.SSEEventExecutionComplete,
				Data:  toExecutionItem(*execution),
			}
			eventJSON, _ := json.Marshal(completeEvent)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", dto.SSEEventExecutionComplete, eventJSON)
		}

		// Update execution
		completedTime := time.Now()
		execution.CompletedAt = &completedTime
		execution.UpdatedAt = &completedTime
		execution.TotalLatencyMs = completedTime.Sub(startTime).Milliseconds()

		execModel := models.GetExecutionModel()
		filter := bson.M{execModel.IdKey: execution.ID}
		update := bson.M{"$set": execution}
		database.UpdateOne(context.Background(), execModel, filter, update)

		w.Flush()
	})

	return nil
}

func executeFlowAsync(execution *models.Execution, agent *models.Agent, flow *models.PromptFlow, initialMessages []services.ChatMessage, database db.DB) {
	startTime := time.Now()

	// Get executor service
	executor := services.NewPromptFlowExecutorService(database)

	// Execute flow (use background context)
	// Note: In production, consider using a context with timeout
	ctx := context.Background()

	err := executor.ExecuteFlow(ctx, execution, agent, flow, initialMessages, nil)

	// Update execution status
	if err != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = err.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
	}

	completedTime := time.Now()
	execution.CompletedAt = &completedTime
	execution.UpdatedAt = &completedTime
	execution.TotalLatencyMs = completedTime.Sub(startTime).Milliseconds()

	// Save to database
	execModel := models.GetExecutionModel()
	filter := bson.M{execModel.IdKey: execution.ID}
	update := bson.M{"$set": execution}
	database.UpdateOne(ctx, execModel, filter, update)
}

func loadPromptFlow(c *fiber.Ctx, database db.DB, flowID string) (*models.PromptFlow, error) {
	flowModel := models.GetPromptFlowModel()
	filter := bson.M{flowModel.IdKey: flowID}

	var flow models.PromptFlow
	if err := database.FindOne(c.Context(), flowModel, filter).Decode(&flow); err != nil {
		return nil, err
	}

	return &flow, nil
}

func toExecutionItem(execution models.Execution) dto.ExecutionItem {
	return dto.ExecutionItem{
		ID:             execution.ID,
		AgentID:        execution.AgentID,
		PromptFlowID:   execution.PromptFlowID,
		Status:         string(execution.Status),
		Input:          execution.Input,
		FinalOutput:    execution.FinalOutput,
		Steps:          toExecutionSteps(execution.Steps),
		Error:          execution.Error,
		CreatedAt:      execution.CreatedAt,
		UpdatedAt:      execution.UpdatedAt,
		CompletedAt:    execution.CompletedAt,
		TotalLatencyMs: execution.TotalLatencyMs,
		Metadata:       execution.Metadata,
	}
}

func toExecutionSteps(steps []models.ExecutionStep) []dto.ExecutionStepItem {
	items := make([]dto.ExecutionStepItem, len(steps))
	for i, step := range steps {
		items[i] = dto.ExecutionStepItem{
			ID:               step.ID,
			StageID:          step.StageID,
			StageType:        step.StageType,
			StageName:        step.StageName,
			StartedAt:        step.StartedAt,
			CompletedAt:      step.CompletedAt,
			Status:           step.Status,
			InputMessages:    toExecutionMessages(step.InputMessages),
			OutputMessage:    toExecutionMessagePtr(step.OutputMessage),
			LLMCalls:         toExecutionLLMMetadata(step.LLMCalls),
			ToolCalls:        toExecutionToolCallResults(step.ToolCalls),
			RetrievedContext: step.RetrievedContext,
			NextStageID:      step.NextStageID,
			TransitionReason: step.TransitionReason,
			Error:            step.Error,
			Metadata:         step.Metadata,
		}
	}
	return items
}

func toExecutionMessages(messages []models.ExecutionMessage) []dto.ExecutionMessageItem {
	items := make([]dto.ExecutionMessageItem, len(messages))
	for i, msg := range messages {
		items[i] = dto.ExecutionMessageItem{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  toExecutionMessageToolCalls(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}
	}
	return items
}

func toExecutionMessagePtr(msg *models.ExecutionMessage) *dto.ExecutionMessageItem {
	if msg == nil {
		return nil
	}
	return &dto.ExecutionMessageItem{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCalls:  toExecutionMessageToolCalls(msg.ToolCalls),
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
	}
}

func toExecutionLLMMetadata(metadata []models.ExecutionLLMMetadata) []dto.ExecutionLLMMetadataItem {
	items := make([]dto.ExecutionLLMMetadataItem, len(metadata))
	for i, m := range metadata {
		items[i] = dto.ExecutionLLMMetadataItem{
			ProviderID:       m.ProviderID,
			Provider:         m.Provider,
			Model:            m.Model,
			Temperature:      m.Temperature,
			MaxTokens:        m.MaxTokens,
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			TotalTokens:      m.TotalTokens,
			LatencyMs:        m.LatencyMs,
			FinishReason:     m.FinishReason,
		}
	}
	return items
}

func toExecutionMessageToolCalls(toolCalls []models.ExecutionToolCall) []dto.ExecutionToolCallItem {
	items := make([]dto.ExecutionToolCallItem, len(toolCalls))
	for i, tc := range toolCalls {
		items[i] = dto.ExecutionToolCallItem{
			ID:        tc.ID,
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
		}
	}
	return items
}

func toExecutionToolCallResults(toolCalls []models.ExecutionToolCall) []dto.ExecutionToolCallResult {
	items := make([]dto.ExecutionToolCallResult, len(toolCalls))
	for i, tc := range toolCalls {
		items[i] = dto.ExecutionToolCallResult{
			ID:          tc.ID,
			ToolID:      tc.ToolID,
			ToolName:    tc.ToolName,
			ToolType:    tc.ToolType,
			Arguments:   tc.Arguments,
			Result:      tc.Result,
			Error:       tc.Error,
			StartedAt:   tc.StartedAt,
			CompletedAt: tc.CompletedAt,
			LatencyMs:   tc.LatencyMs,
		}
	}
	return items
}
