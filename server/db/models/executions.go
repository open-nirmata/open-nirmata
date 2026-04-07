package models

import (
	"open-nirmata/dto"
	"time"
)

// ExecutionStatus represents the current state of an execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionMessage represents a message in the conversation
type ExecutionMessage struct {
	Role       string                 `bson:"role"`                   // "system", "user", "assistant", "tool"
	Content    string                 `bson:"content,omitempty"`      // message content
	ToolCalls  []ExecutionToolCall    `bson:"tool_calls,omitempty"`   // tool calls from assistant
	ToolCallID string                 `bson:"tool_call_id,omitempty"` // for tool role messages
	Name       string                 `bson:"name,omitempty"`         // tool name for tool messages
	Metadata   map[string]interface{} `bson:"metadata,omitempty"`     // additional metadata
}

// ExecutionToolCall represents a tool call made by the LLM
type ExecutionToolCall struct {
	ID          string                 `bson:"id"`                     // unique tool call ID
	ToolID      string                 `bson:"tool_id,omitempty"`      // reference to tool in DB
	ToolName    string                 `bson:"tool_name"`              // name of the tool
	ToolType    string                 `bson:"tool_type,omitempty"`    // "mcp", "http", "llm"
	Arguments   map[string]interface{} `bson:"arguments"`              // tool input parameters
	Result      string                 `bson:"result,omitempty"`       // tool execution result
	Error       string                 `bson:"error,omitempty"`        // error if tool failed
	StartedAt   *time.Time             `bson:"started_at,omitempty"`   // when tool execution started
	CompletedAt *time.Time             `bson:"completed_at,omitempty"` // when tool execution completed
	LatencyMs   int64                  `bson:"latency_ms,omitempty"`   // execution time in milliseconds
}

// ExecutionLLMMetadata contains metadata about an LLM call
type ExecutionLLMMetadata struct {
	ProviderID       string   `bson:"provider_id"`                 // LLM provider ID
	Provider         string   `bson:"provider"`                    // provider type (openai, anthropic, etc.)
	Model            string   `bson:"model"`                       // model name
	Temperature      *float64 `bson:"temperature,omitempty"`       // temperature setting
	MaxTokens        *int     `bson:"max_tokens,omitempty"`        // max tokens setting
	PromptTokens     int      `bson:"prompt_tokens,omitempty"`     // tokens in prompt
	CompletionTokens int      `bson:"completion_tokens,omitempty"` // tokens in completion
	TotalTokens      int      `bson:"total_tokens,omitempty"`      // total tokens used
	LatencyMs        int64    `bson:"latency_ms,omitempty"`        // LLM call latency in milliseconds
	FinishReason     string   `bson:"finish_reason,omitempty"`     // stop, length, tool_calls, etc.
}

// ExecutionStep represents the execution of a single stage in the prompt flow
type ExecutionStep struct {
	ID               string                  `bson:"id"`                          // unique step ID
	StageID          string                  `bson:"stage_id"`                    // stage ID from prompt flow
	StageName        string                  `bson:"stage_name"`                  // stage name
	StageType        dto.PromptFlowStageType `bson:"stage_type"`                  // chat, tool, retrieval, router
	StartedAt        *time.Time              `bson:"started_at"`                  // when step started
	CompletedAt      *time.Time              `bson:"completed_at,omitempty"`      // when step completed
	Status           string                  `bson:"status"`                      // running, completed, failed
	InputMessages    []ExecutionMessage      `bson:"input_messages,omitempty"`    // messages going into this stage
	OutputMessage    *ExecutionMessage       `bson:"output_message,omitempty"`    // message produced by this stage
	LLMCalls         []ExecutionLLMMetadata  `bson:"llm_calls,omitempty"`         // LLM calls made in this stage
	ToolCalls        []ExecutionToolCall     `bson:"tool_calls,omitempty"`        // tools executed in this stage
	RetrievedContext []string                `bson:"retrieved_context,omitempty"` // knowledge retrieved
	NextStageID      string                  `bson:"next_stage_id,omitempty"`     // which stage to transition to
	TransitionReason string                  `bson:"transition_reason,omitempty"` // why this transition was chosen
	Error            string                  `bson:"error,omitempty"`             // error if step failed
	Metadata         map[string]interface{}  `bson:"metadata,omitempty"`          // additional step metadata
}

// Execution represents a complete execution of a prompt flow
type Execution struct {
	ID             string                 `bson:"id"`                         // unique execution ID
	AgentID        string                 `bson:"agent_id"`                   // agent that was executed
	PromptFlowID   string                 `bson:"prompt_flow_id"`             // prompt flow that was executed
	Status         ExecutionStatus        `bson:"status"`                     // running, completed, failed, cancelled
	Input          map[string]interface{} `bson:"input"`                      // initial input (message or messages array)
	Steps          []ExecutionStep        `bson:"steps,omitempty"`            // execution steps in order
	FinalOutput    string                 `bson:"final_output,omitempty"`     // final response
	Error          string                 `bson:"error,omitempty"`            // error if execution failed
	CreatedAt      *time.Time             `bson:"created_at"`                 // when execution started
	UpdatedAt      *time.Time             `bson:"updated_at,omitempty"`       // last update time
	CompletedAt    *time.Time             `bson:"completed_at,omitempty"`     // when execution finished
	TotalLatencyMs int64                  `bson:"total_latency_ms,omitempty"` // total execution time
	Metadata       map[string]interface{} `bson:"metadata,omitempty"`         // additional execution metadata
}

// ExecutionModel provides collection and field mappings for execution documents
type ExecutionModel struct {
	openNirmata
	IdKey             string
	AgentIDKey        string
	PromptFlowIDKey   string
	StatusKey         string
	InputKey          string
	StepsKey          string
	FinalOutputKey    string
	ErrorKey          string
	CreatedAtKey      string
	UpdatedAtKey      string
	CompletedAtKey    string
	TotalLatencyMsKey string
	MetadataKey       string
}

func (e ExecutionModel) Name() string {
	return "executions"
}

func GetExecutionModel() ExecutionModel {
	return ExecutionModel{
		IdKey:             "id",
		AgentIDKey:        "agent_id",
		PromptFlowIDKey:   "prompt_flow_id",
		StatusKey:         "status",
		InputKey:          "input",
		StepsKey:          "steps",
		FinalOutputKey:    "final_output",
		ErrorKey:          "error",
		CreatedAtKey:      "created_at",
		UpdatedAtKey:      "updated_at",
		CompletedAtKey:    "completed_at",
		TotalLatencyMsKey: "total_latency_ms",
		MetadataKey:       "metadata",
	}
}
