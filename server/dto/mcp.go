package dto

type TestMCPToolRequest struct {
	Config         *ToolConfig `json:"config"`
	TimeoutSeconds *int        `json:"timeout_seconds,omitempty"`
}

type MCPServerInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type MCPDiscoveredTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

type TestMCPToolResult struct {
	Transport  string              `json:"transport"`
	ServerURL  string              `json:"server_url,omitempty"`
	ServerInfo *MCPServerInfo      `json:"server_info,omitempty"`
	Tools      []MCPDiscoveredTool `json:"tools"`
	Count      int                 `json:"count"`
}

type TestMCPToolResponse struct {
	Success bool               `json:"success"`
	Data    *TestMCPToolResult `json:"data,omitempty"`
	Message string             `json:"message,omitempty"`
}
