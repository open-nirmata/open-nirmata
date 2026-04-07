package dto

import "time"

type ToolType string

type ToolProvider string

const (
	ToolTypeMCP  ToolType = "mcp"
	ToolTypeHTTP ToolType = "http"
	ToolTypeLLM  ToolType = "llm"
)

const (
	ToolProviderOpenAI     ToolProvider = "openai"
	ToolProviderOllama     ToolProvider = "ollama"
	ToolProviderAnthropic  ToolProvider = "anthropic"
	ToolProviderGroq       ToolProvider = "groq"
	ToolProviderOpenRouter ToolProvider = "openrouter"
	ToolProviderGemini     ToolProvider = "gemini"
)

type ToolConfig struct {
	// HTTP tool fields.
	URL             string            `json:"url,omitempty" bson:"url,omitempty"`
	Method          string            `json:"method,omitempty" bson:"method,omitempty"`
	PayloadTemplate string            `json:"payload_template,omitempty" bson:"payload_template,omitempty"`
	Headers         map[string]string `json:"headers,omitempty" bson:"headers,omitempty"`
	QueryParams     map[string]string `json:"query_params,omitempty" bson:"query_params,omitempty"`
	TimeoutSeconds  *int              `json:"timeout_seconds,omitempty" bson:"timeout_seconds,omitempty"`

	// MCP tool fields.
	Transport       string                 `json:"transport,omitempty" bson:"transport,omitempty"`
	Command         string                 `json:"command,omitempty" bson:"command,omitempty"`
	Args            []string               `json:"args,omitempty" bson:"args,omitempty"`
	Env             map[string]string      `json:"env,omitempty" bson:"env,omitempty"`
	ServerURL       string                 `json:"server_url,omitempty" bson:"server_url,omitempty"`
	InputSchema     map[string]interface{} `json:"input_schema,omitempty" bson:"input_schema,omitempty"`
	Annotations     map[string]interface{} `json:"annotations,omitempty" bson:"annotations,omitempty"`
	ServerInfo      *MCPServerInfo         `json:"server_info,omitempty" bson:"server_info,omitempty"`
	LastRefreshedAt *time.Time             `json:"last_refreshed_at,omitempty" bson:"last_refreshed_at,omitempty"`
}

type CreateToolRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Provider    string                 `json:"provider,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Config      *ToolConfig            `json:"config,omitempty"`
	Auth        map[string]interface{} `json:"auth,omitempty"`
}

type UpdateToolRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Type        *string                 `json:"type,omitempty"`
	Provider    *string                 `json:"provider,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	Tags        *[]string               `json:"tags,omitempty"`
	Config      *ToolConfig             `json:"config,omitempty"`
	Auth        *map[string]interface{} `json:"auth,omitempty"`
}

type RefreshToolRequest struct {
	TimeoutSeconds *int `json:"timeout_seconds,omitempty"`
}

type ToolItem struct {
	Id             string      `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	Provider       string      `json:"provider,omitempty"`
	Description    string      `json:"description,omitempty"`
	Enabled        bool        `json:"enabled"`
	Tags           []string    `json:"tags,omitempty"`
	Config         *ToolConfig `json:"config,omitempty"`
	AuthConfigured bool        `json:"auth_configured"`
	CreatedAt      *time.Time  `json:"created_at,omitempty"`
	UpdatedAt      *time.Time  `json:"updated_at,omitempty"`
}

type ToolResponse struct {
	Success bool      `json:"success"`
	Data    *ToolItem `json:"data,omitempty"`
	Message string    `json:"message,omitempty"`
}

type ToolListResponse struct {
	Success bool       `json:"success"`
	Data    []ToolItem `json:"data"`
	Count   int        `json:"count"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
