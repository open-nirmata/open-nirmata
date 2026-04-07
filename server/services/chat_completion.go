package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"open-nirmata/db/models"
)

const defaultChatCompletionTimeout = 60 * time.Second

// ChatCompletionService handles LLM chat completion requests
type ChatCompletionService struct {
	client *http.Client
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

// ChatToolCall represents a tool call in a message
type ChatToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type,omitempty"` // "function"
	Function ChatToolCallFunction `json:"function,omitempty"`
}

// ChatToolCallFunction represents the function details of a tool call
type ChatToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ChatCompletionRequest represents a request for chat completion
type ChatCompletionRequest struct {
	Provider     *models.LLMProvider
	Model        string
	Messages     []ChatMessage
	SystemPrompt string
	Temperature  *float64
	MaxTokens    *int
	Tools        []ChatTool
	Stream       bool
}

// ChatTool represents a tool that can be called by the LLM
type ChatTool struct {
	Type     string           `json:"type"` // "function"
	Function ChatToolFunction `json:"function"`
}

// ChatToolFunction represents the function definition of a tool
type ChatToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatCompletionResponse represents the response from chat completion
type ChatCompletionResponse struct {
	Content          string
	ToolCalls        []ChatToolCall
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Model            string
}

// ChatCompletionStreamChunk represents a chunk in streaming response
type ChatCompletionStreamChunk struct {
	Content      string
	ToolCall     *ChatToolCall
	FinishReason string
	Done         bool
}

func NewChatCompletionService() *ChatCompletionService {
	return &ChatCompletionService{
		client: &http.Client{},
	}
}

// ChatCompletion performs a non-streaming chat completion
func (s *ChatCompletionService) ChatCompletion(ctx context.Context, req *ChatCompletionRequest, timeout time.Duration) (*ChatCompletionResponse, error) {
	if req == nil || req.Provider == nil {
		return nil, fmt.Errorf("request and provider are required")
	}

	if timeout <= 0 {
		timeout = defaultChatCompletionTimeout
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	provider := strings.ToLower(strings.TrimSpace(req.Provider.Provider))
	switch provider {
	case "openai", "groq", "openrouter":
		return s.chatCompletionOpenAIStyle(timedCtx, req, provider)
	case "anthropic":
		return s.chatCompletionAnthropic(timedCtx, req)
	case "gemini":
		return s.chatCompletionGemini(timedCtx, req)
	case "ollama":
		return s.chatCompletionOllama(timedCtx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ChatCompletionStream performs a streaming chat completion
func (s *ChatCompletionService) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest, timeout time.Duration, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	if req == nil || req.Provider == nil {
		return nil, fmt.Errorf("request and provider are required")
	}

	if timeout <= 0 {
		timeout = defaultChatCompletionTimeout
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	provider := strings.ToLower(strings.TrimSpace(req.Provider.Provider))
	switch provider {
	case "openai", "groq", "openrouter":
		return s.chatCompletionStreamOpenAIStyle(timedCtx, req, provider, callback)
	case "anthropic":
		return s.chatCompletionStreamAnthropic(timedCtx, req, callback)
	case "gemini":
		return s.chatCompletionStreamGemini(timedCtx, req, callback)
	case "ollama":
		return s.chatCompletionStreamOllama(timedCtx, req, callback)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// OpenAI-style providers (OpenAI, Groq, OpenRouter)
func (s *ChatCompletionService) chatCompletionOpenAIStyle(ctx context.Context, req *ChatCompletionRequest, provider string) (*ChatCompletionResponse, error) {
	endpoint := s.getOpenAIStyleEndpoint(req.Provider, provider)
	headers := s.getOpenAIStyleHeaders(req.Provider, provider)

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": s.formatMessagesOpenAI(req.Messages, req.SystemPrompt),
		"stream":   false,
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}

	var response map[string]interface{}
	if err := s.postJSON(ctx, endpoint, headers, payload, &response); err != nil {
		return nil, err
	}

	return s.parseOpenAIStyleResponse(response)
}

func (s *ChatCompletionService) chatCompletionStreamOpenAIStyle(ctx context.Context, req *ChatCompletionRequest, provider string, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	endpoint := s.getOpenAIStyleEndpoint(req.Provider, provider)
	headers := s.getOpenAIStyleHeaders(req.Provider, provider)

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": s.formatMessagesOpenAI(req.Messages, req.SystemPrompt),
		"stream":   true,
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}

	return s.streamOpenAIStyle(ctx, endpoint, headers, payload, callback)
}

func (s *ChatCompletionService) getOpenAIStyleEndpoint(provider *models.LLMProvider, providerType string) string {
	baseURL := strings.TrimSpace(provider.BaseURL)
	switch providerType {
	case "openai":
		return resolveEndpoint(baseURL, "https://api.openai.com", "/v1/chat/completions")
	case "groq":
		return resolveEndpoint(baseURL, "https://api.groq.com/openai", "/v1/chat/completions")
	case "openrouter":
		return resolveEndpoint(baseURL, "https://openrouter.ai/api", "/v1/chat/completions")
	default:
		return resolveEndpoint(baseURL, baseURL, "/v1/chat/completions")
	}
}

func (s *ChatCompletionService) getOpenAIStyleHeaders(provider *models.LLMProvider, providerType string) map[string]string {
	apiKey := s.extractAPIKey(provider.Auth)
	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}

	if providerType == "openai" {
		if org := strings.TrimSpace(provider.Organization); org != "" {
			headers["OpenAI-Organization"] = org
		}
		if proj := strings.TrimSpace(provider.ProjectID); proj != "" {
			headers["OpenAI-Project"] = proj
		}
	}

	return headers
}

func (s *ChatCompletionService) formatMessagesOpenAI(messages []ChatMessage, systemPrompt string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages)+1)

	if systemPrompt != "" {
		result = append(result, map[string]interface{}{
			"role":    "system",
			"content": systemPrompt,
		})
	}

	for _, msg := range messages {
		formatted := map[string]interface{}{
			"role": msg.Role,
		}

		if msg.Content != "" {
			formatted["content"] = msg.Content
		}

		if len(msg.ToolCalls) > 0 {
			formatted["tool_calls"] = msg.ToolCalls
		}

		if msg.ToolCallID != "" {
			formatted["tool_call_id"] = msg.ToolCallID
		}

		if msg.Name != "" {
			formatted["name"] = msg.Name
		}

		result = append(result, formatted)
	}

	return result
}

func (s *ChatCompletionService) parseOpenAIStyleResponse(response map[string]interface{}) (*ChatCompletionResponse, error) {
	choices := toInterfaceSlice(response["choices"])
	if len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := toLooseMap(choices[0])
	message := toLooseMap(choice["message"])

	result := &ChatCompletionResponse{
		Content:      stringValue(message["content"]),
		FinishReason: stringValue(choice["finish_reason"]),
		Model:        stringValue(response["model"]),
	}

	// Parse tool calls
	toolCalls := toInterfaceSlice(message["tool_calls"])
	for _, tc := range toolCalls {
		toolCall := toLooseMap(tc)
		function := toLooseMap(toolCall["function"])

		var args map[string]interface{}
		if argsStr := stringValue(function["arguments"]); argsStr != "" {
			json.Unmarshal([]byte(argsStr), &args)
		}

		result.ToolCalls = append(result.ToolCalls, ChatToolCall{
			ID:   stringValue(toolCall["id"]),
			Type: stringValue(toolCall["type"]),
			Function: ChatToolCallFunction{
				Name:      stringValue(function["name"]),
				Arguments: args,
			},
		})
	}

	// Parse usage
	usage := toLooseMap(response["usage"])
	result.PromptTokens = intValue(usage["prompt_tokens"])
	result.CompletionTokens = intValue(usage["completion_tokens"])
	result.TotalTokens = intValue(usage["total_tokens"])

	return result, nil
}

func (s *ChatCompletionService) streamOpenAIStyle(ctx context.Context, endpoint string, headers map[string]string, payload map[string]interface{}, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, sanitizeErrorBody(body))
	}

	// Accumulate response
	var fullContent strings.Builder
	var toolCalls []ChatToolCall
	var finishReason string
	var model string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if model == "" {
			model = stringValue(chunk["model"])
		}

		choices := toInterfaceSlice(chunk["choices"])
		if len(choices) == 0 {
			continue
		}

		choice := toLooseMap(choices[0])
		delta := toLooseMap(choice["delta"])

		// Handle content - don't use stringValue as it trims spaces
		var content string
		if c, ok := delta["content"].(string); ok {
			content = c
		}
		if content != "" {
			fullContent.WriteString(content)
			if err := callback(&ChatCompletionStreamChunk{Content: content}); err != nil {
				return nil, err
			}
		}

		// Handle tool calls
		deltaToolCalls := toInterfaceSlice(delta["tool_calls"])
		for _, tc := range deltaToolCalls {
			toolCall := toLooseMap(tc)
			function := toLooseMap(toolCall["function"])

			var args map[string]interface{}
			if argsStr := stringValue(function["arguments"]); argsStr != "" {
				json.Unmarshal([]byte(argsStr), &args)
			}

			call := ChatToolCall{
				ID:   stringValue(toolCall["id"]),
				Type: stringValue(toolCall["type"]),
				Function: ChatToolCallFunction{
					Name:      stringValue(function["name"]),
					Arguments: args,
				},
			}
			toolCalls = append(toolCalls, call)

			if err := callback(&ChatCompletionStreamChunk{ToolCall: &call}); err != nil {
				return nil, err
			}
		}

		if fr := stringValue(choice["finish_reason"]); fr != "" {
			finishReason = fr
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	if err := callback(&ChatCompletionStreamChunk{Done: true, FinishReason: finishReason}); err != nil {
		return nil, err
	}

	return &ChatCompletionResponse{
		Content:      fullContent.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Model:        model,
	}, nil
}

// Anthropic implementation
func (s *ChatCompletionService) chatCompletionAnthropic(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Anthropic uses a different format - system is separate, messages are user/assistant only
	endpoint := resolveEndpoint(req.Provider.BaseURL, "https://api.anthropic.com", "/v1/messages")
	apiKey := s.extractAPIKey(req.Provider.Auth)

	headers := map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": s.formatMessagesAnthropic(req.Messages),
		"stream":   false,
	}

	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}

	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	} else {
		payload["max_tokens"] = 4096 // Anthropic requires max_tokens
	}

	if len(req.Tools) > 0 {
		payload["tools"] = s.convertToolsToAnthropic(req.Tools)
	}

	var response map[string]interface{}
	if err := s.postJSON(ctx, endpoint, headers, payload, &response); err != nil {
		return nil, err
	}

	return s.parseAnthropicResponse(response)
}

func (s *ChatCompletionService) chatCompletionStreamAnthropic(ctx context.Context, req *ChatCompletionRequest, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	// Similar to non-streaming but with stream: true
	// Implementation similar to OpenAI streaming but with Anthropic's format
	return nil, fmt.Errorf("anthropic streaming not yet implemented")
}

func (s *ChatCompletionService) formatMessagesAnthropic(messages []ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == "system" {
			continue // System handled separately
		}

		formatted := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}

		result = append(result, formatted)
	}

	return result
}

func (s *ChatCompletionService) convertToolsToAnthropic(tools []ChatTool) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		})
	}
	return result
}

func (s *ChatCompletionService) parseAnthropicResponse(response map[string]interface{}) (*ChatCompletionResponse, error) {
	result := &ChatCompletionResponse{
		Model:        stringValue(response["model"]),
		FinishReason: stringValue(response["stop_reason"]),
	}

	// Content is an array in Anthropic
	content := toInterfaceSlice(response["content"])
	for _, c := range content {
		contentBlock := toLooseMap(c)
		if contentBlock["type"] == "text" {
			result.Content += stringValue(contentBlock["text"])
		}
	}

	// Usage
	usage := toLooseMap(response["usage"])
	result.PromptTokens = intValue(usage["input_tokens"])
	result.CompletionTokens = intValue(usage["output_tokens"])
	result.TotalTokens = result.PromptTokens + result.CompletionTokens

	return result, nil
}

// Gemini implementation (simplified)
func (s *ChatCompletionService) chatCompletionGemini(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	return nil, fmt.Errorf("gemini chat completion not yet implemented")
}

func (s *ChatCompletionService) chatCompletionStreamGemini(ctx context.Context, req *ChatCompletionRequest, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	return nil, fmt.Errorf("gemini streaming not yet implemented")
}

// Ollama implementation (simplified)
func (s *ChatCompletionService) chatCompletionOllama(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	endpoint := resolveEndpoint(req.Provider.BaseURL, req.Provider.BaseURL, "/api/chat")

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": s.formatMessagesOpenAI(req.Messages, req.SystemPrompt),
		"stream":   false,
	}

	if req.Temperature != nil {
		payload["options"] = map[string]interface{}{
			"temperature": *req.Temperature,
		}
	}

	var response map[string]interface{}
	if err := s.postJSON(ctx, endpoint, nil, payload, &response); err != nil {
		return nil, err
	}

	message := toLooseMap(response["message"])
	return &ChatCompletionResponse{
		Content:      stringValue(message["content"]),
		FinishReason: stringValue(response["done_reason"]),
		Model:        stringValue(response["model"]),
	}, nil
}

func (s *ChatCompletionService) chatCompletionStreamOllama(ctx context.Context, req *ChatCompletionRequest, callback func(*ChatCompletionStreamChunk) error) (*ChatCompletionResponse, error) {
	endpoint := resolveEndpoint(req.Provider.BaseURL, req.Provider.BaseURL, "/api/chat")

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": s.formatMessagesOpenAI(req.Messages, req.SystemPrompt),
		"stream":   true,
	}

	if req.Temperature != nil {
		payload["options"] = map[string]interface{}{
			"temperature": *req.Temperature,
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, sanitizeErrorBody(body))
	}

	// Accumulate response
	var fullContent strings.Builder
	var finishReason string
	var model string
	var promptTokens, completionTokens int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if model == "" {
			model = stringValue(chunk["model"])
		}

		// Check if done
		done := chunk["done"] == true

		if !done {
			// Extract content from message
			message := toLooseMap(chunk["message"])
			// Don't use stringValue here as it trims spaces, which removes space-only chunks
			var content string
			if c, ok := message["content"].(string); ok {
				content = c
			}
			if content != "" {
				fullContent.WriteString(content)
				if err := callback(&ChatCompletionStreamChunk{Content: content}); err != nil {
					return nil, err
				}
			}
		} else {
			// Final chunk contains usage info
			finishReason = stringValue(chunk["done_reason"])

			// Ollama returns prompt_eval_count and eval_count
			if evalCount := chunk["eval_count"]; evalCount != nil {
				completionTokens = intValue(evalCount)
			}
			if promptEvalCount := chunk["prompt_eval_count"]; promptEvalCount != nil {
				promptTokens = intValue(promptEvalCount)
			}

			if err := callback(&ChatCompletionStreamChunk{Done: true, FinishReason: finishReason}); err != nil {
				return nil, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	return &ChatCompletionResponse{
		Content:          fullContent.String(),
		FinishReason:     finishReason,
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}, nil
}

// Helper methods
func (s *ChatCompletionService) extractAPIKey(auth map[string]interface{}) string {
	if auth == nil {
		return ""
	}
	if key, ok := auth["api_key"].(string); ok {
		return key
	}
	return ""
}

func (s *ChatCompletionService) postJSON(ctx context.Context, endpoint string, headers map[string]string, payload interface{}, target interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("provider returned %s: %s", resp.Status, sanitizeErrorBody(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
