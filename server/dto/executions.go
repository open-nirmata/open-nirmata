package dto

import (
	"time"
)

// ExecuteAgentMode specifies how the agent execution should be handled
type ExecuteAgentMode string

const (
	ExecuteAgentModeSync  ExecuteAgentMode = "sync"  // synchronous execution, wait for result
	ExecuteAgentModeAsync ExecuteAgentMode = "async" // asynchronous execution, return job ID
)

// ExecuteAgentRequest represents a request to execute an agent
type ExecuteAgentRequest struct {
	// Single message input (convenience mode)
	Message string `json:"message,omitempty"`

	// Full conversation history mode
	Messages []ExecutionMessageItem `json:"messages,omitempty"`

	// Execution configuration
	Stream   bool                   `json:"stream,omitempty"`   // enable SSE streaming
	Metadata map[string]interface{} `json:"metadata,omitempty"` // additional metadata
}

// ExecutionMessageItem represents a message in the conversation
type ExecutionMessageItem struct {
	Role       string                  `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string                  `json:"content,omitempty"`      // message content
	ToolCalls  []ExecutionToolCallItem `json:"tool_calls,omitempty"`   // tool calls from assistant
	ToolCallID string                  `json:"tool_call_id,omitempty"` // for tool role messages
	Name       string                  `json:"name,omitempty"`         // tool name for tool messages
}

// ExecutionToolCallItem represents a tool call in messages
type ExecutionToolCallItem struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ExecutionToolCallResult represents a completed tool call with its result
type ExecutionToolCallResult struct {
	ID          string                 `json:"id"`
	ToolID      string                 `json:"tool_id,omitempty"`
	ToolName    string                 `json:"tool_name"`
	ToolType    string                 `json:"tool_type,omitempty"`
	Arguments   map[string]interface{} `json:"arguments"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	LatencyMs   int64                  `json:"latency_ms,omitempty"`
}

// ExecutionLLMMetadataItem represents metadata about an LLM call
type ExecutionLLMMetadataItem struct {
	ProviderID       string   `json:"provider_id"`
	Provider         string   `json:"provider"`
	Model            string   `json:"model"`
	Temperature      *float64 `json:"temperature,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	PromptTokens     int      `json:"prompt_tokens,omitempty"`
	CompletionTokens int      `json:"completion_tokens,omitempty"`
	TotalTokens      int      `json:"total_tokens,omitempty"`
	LatencyMs        int64    `json:"latency_ms,omitempty"`
	FinishReason     string   `json:"finish_reason,omitempty"`
}

// ExecutionStepItem represents a single step in the execution
type ExecutionStepItem struct {
	ID               string                     `json:"id"`
	StageID          string                     `json:"stage_id"`
	StageName        string                     `json:"stage_name"`
	StageType        PromptFlowStageType        `json:"stage_type"`
	StartedAt        *time.Time                 `json:"started_at"`
	CompletedAt      *time.Time                 `json:"completed_at,omitempty"`
	Status           string                     `json:"status"`
	InputMessages    []ExecutionMessageItem     `json:"input_messages,omitempty"`
	OutputMessage    *ExecutionMessageItem      `json:"output_message,omitempty"`
	LLMCalls         []ExecutionLLMMetadataItem `json:"llm_calls,omitempty"`
	ToolCalls        []ExecutionToolCallResult  `json:"tool_calls,omitempty"`
	RetrievedContext []string                   `json:"retrieved_context,omitempty"`
	NextStageID      string                     `json:"next_stage_id,omitempty"`
	TransitionReason string                     `json:"transition_reason,omitempty"`
	Error            string                     `json:"error,omitempty"`
	Metadata         map[string]interface{}     `json:"metadata,omitempty"`
}

// ExecutionItem represents an execution with its details
type ExecutionItem struct {
	ID             string                 `json:"id"`
	AgentID        string                 `json:"agent_id"`
	PromptFlowID   string                 `json:"prompt_flow_id"`
	Status         string                 `json:"status"`
	Input          map[string]interface{} `json:"input"`
	Steps          []ExecutionStepItem    `json:"steps,omitempty"`
	FinalOutput    string                 `json:"final_output,omitempty"`
	Error          string                 `json:"error,omitempty"`
	CreatedAt      *time.Time             `json:"created_at"`
	UpdatedAt      *time.Time             `json:"updated_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	TotalLatencyMs int64                  `json:"total_latency_ms,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteAgentResponse represents the response from an agent execution
type ExecuteAgentResponse struct {
	Success bool           `json:"success"`
	Data    *ExecutionItem `json:"data,omitempty"`   // for sync mode: complete execution
	JobID   string         `json:"job_id,omitempty"` // for async mode: execution ID to poll
	Message string         `json:"message,omitempty"`
}

// GetExecutionResponse represents the response when fetching an execution
type GetExecutionResponse struct {
	Success bool           `json:"success"`
	Data    *ExecutionItem `json:"data,omitempty"`
	Message string         `json:"message,omitempty"`
}

// ListExecutionsResponse represents a list of executions
type ListExecutionsResponse struct {
	Success bool            `json:"success"`
	Data    []ExecutionItem `json:"data"`
	Count   int             `json:"count"`
}

// SSE Event types for streaming
const (
	SSEEventStageStart        = "stage_start"
	SSEEventLLMToken          = "llm_token"
	SSEEventLLMComplete       = "llm_complete"
	SSEEventToolCall          = "tool_call"
	SSEEventToolResult        = "tool_result"
	SSEEventStageComplete     = "stage_complete"
	SSEEventExecutionComplete = "execution_complete"
	SSEEventError             = "error"
)

// SSEEvent represents a server-sent event during streaming
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// SSEStageStartData contains data for stage_start event
type SSEStageStartData struct {
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	StageType string `json:"stage_type"`
}

// SSELLMTokenData contains data for llm_token event
type SSELLMTokenData struct {
	Token string `json:"token"`
	Delta string `json:"delta"` // for compatibility
}

// SSEToolCallData contains data for tool_call event
type SSEToolCallData struct {
	ToolCallID string                 `json:"tool_call_id"`
	ToolName   string                 `json:"tool_name"`
	Arguments  map[string]interface{} `json:"arguments"`
}

// SSEToolResultData contains data for tool_result event
type SSEToolResultData struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Result     string `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
}

// SSEStageCompleteData contains data for stage_complete event
type SSEStageCompleteData struct {
	StageID     string `json:"stage_id"`
	NextStageID string `json:"next_stage_id,omitempty"`
	Complete    bool   `json:"complete"` // true if this was the final stage
}

// SSEErrorData contains data for error event
type SSEErrorData struct {
	Error   string `json:"error"`
	StageID string `json:"stage_id,omitempty"`
}
