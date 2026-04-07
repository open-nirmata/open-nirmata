package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"open-nirmata/db/models"
	"open-nirmata/dto"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const defaultToolExecutionTimeout = 30 * time.Second

// ToolExecutorService handles tool execution for MCP and HTTP tools
type ToolExecutorService struct {
	mcpService *MCPService
	client     *http.Client
}

// ToolExecutionRequest represents a request to execute a tool
type ToolExecutionRequest struct {
	Tool      *models.Tool
	ToolName  string                 // for MCP tools, the specific tool name to call
	Arguments map[string]interface{} // tool input parameters
}

// ToolExecutionResult represents the result of tool execution
type ToolExecutionResult struct {
	ToolID      string
	ToolName    string
	ToolType    string
	Arguments   map[string]interface{}
	Result      string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
	LatencyMs   int64
}

func NewToolExecutorService() *ToolExecutorService {
	return &ToolExecutorService{
		mcpService: NewMCPService(),
		client:     &http.Client{},
	}
}

// ExecuteTool executes a tool and returns the result
func (s *ToolExecutorService) ExecuteTool(ctx context.Context, req *ToolExecutionRequest, timeout time.Duration) (*ToolExecutionResult, error) {
	if req == nil || req.Tool == nil {
		return nil, fmt.Errorf("request and tool are required")
	}

	if timeout <= 0 {
		timeout = defaultToolExecutionTimeout
	}

	startTime := time.Now()

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := &ToolExecutionResult{
		ToolID:    req.Tool.Id,
		ToolName:  req.ToolName,
		ToolType:  req.Tool.Type,
		Arguments: req.Arguments,
		StartedAt: startTime,
	}

	toolType := strings.ToLower(strings.TrimSpace(req.Tool.Type))
	var resultStr string
	var err error

	switch toolType {
	case "mcp":
		resultStr, err = s.executeMCPTool(timedCtx, req)
	case "http":
		resultStr, err = s.executeHTTPTool(timedCtx, req)
	default:
		err = fmt.Errorf("unsupported tool type: %s", toolType)
	}

	result.CompletedAt = time.Now()
	result.LatencyMs = result.CompletedAt.Sub(startTime).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Result = resultStr
	return result, nil
}

// ExecuteTools executes multiple tools in sequence
func (s *ToolExecutorService) ExecuteTools(ctx context.Context, requests []*ToolExecutionRequest, timeout time.Duration) ([]*ToolExecutionResult, error) {
	results := make([]*ToolExecutionResult, 0, len(requests))

	for _, req := range requests {
		result, err := s.ExecuteTool(ctx, req, timeout)
		if err != nil {
			// Continue executing other tools even if one fails
			results = append(results, result)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// executeMCPTool executes an MCP tool
func (s *ToolExecutorService) executeMCPTool(ctx context.Context, req *ToolExecutionRequest) (string, error) {
	if req.Tool.Config == nil {
		return "", fmt.Errorf("tool config is required for MCP tools")
	}

	normalized := normalizeMCPToolConfig(req.Tool.Config)
	transport := resolveTransport(normalized)

	switch transport {
	case "stdio":
		return s.executeMCPToolStdio(ctx, req, normalized)
	case "remote":
		return s.executeMCPToolRemote(ctx, req, normalized)
	default:
		return "", fmt.Errorf("unsupported MCP transport: %s", transport)
	}
}

func (s *ToolExecutorService) executeMCPToolStdio(ctx context.Context, req *ToolExecutionRequest, config *dto.ToolConfig) (string, error) {
	client, err := mcpclient.NewStdioMCPClient(config.Command, stringMapToEnv(config.Env), config.Args...)
	if err != nil {
		return "", fmt.Errorf("failed to create stdio MCP client: %w", err)
	}

	return s.callMCPTool(ctx, client, req.ToolName, req.Arguments)
}

func (s *ToolExecutorService) executeMCPToolRemote(ctx context.Context, req *ToolExecutionRequest, config *dto.ToolConfig) (string, error) {
	serverURL := strings.TrimSpace(config.ServerURL)
	if serverURL == "" {
		return "", fmt.Errorf("server_url is required for remote MCP tools")
	}

	timeout := defaultToolExecutionTimeout
	if config.TimeoutSeconds != nil && *config.TimeoutSeconds > 0 {
		timeout = time.Duration(*config.TimeoutSeconds) * time.Second
	}

	var client *mcpclient.Client
	var err error

	if looksLikeSSEEndpoint(serverURL) {
		client, err = newSSEClient(serverURL, config.Headers, timeout)
	} else {
		client, err = newStreamableHTTPClient(serverURL, config.Headers, timeout)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create remote MCP client: %w", err)
	}

	return s.callMCPTool(ctx, client, req.ToolName, req.Arguments)
}

func (s *ToolExecutorService) callMCPTool(ctx context.Context, client *mcpclient.Client, toolName string, arguments map[string]interface{}) (string, error) {
	defer client.Close()

	if err := client.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start MCP client: %w", err)
	}

	if _, err := client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: defaultMCPProtocolVersion,
			ClientInfo: mcp.Implementation{
				Name:    "open-nirmata",
				Version: "latest",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}); err != nil {
		return "", fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	// Call the tool
	callResult, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %q: %w", toolName, err)
	}

	// Parse result
	if callResult.IsError {
		return "", fmt.Errorf("MCP tool returned error: %v", callResult.Content)
	}

	// Convert content to string
	return s.parseMCPContentToString(callResult.Content), nil
}

func (s *ToolExecutorService) parseMCPContentToString(content []mcp.Content) string {
	if len(content) == 0 {
		return ""
	}

	var result strings.Builder
	for _, item := range content {
		// Try to marshal to JSON to extract text fields
		bytes, err := json.Marshal(item)
		if err == nil {
			var contentMap map[string]interface{}
			if json.Unmarshal(bytes, &contentMap) == nil {
				if text, ok := contentMap["text"].(string); ok && text != "" {
					result.WriteString(text)
					result.WriteString("\n")
					continue
				}
			}
		}
		// Fallback: just write the JSON representation
		result.WriteString(string(bytes))
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String())
}

// executeHTTPTool executes an HTTP tool
func (s *ToolExecutorService) executeHTTPTool(ctx context.Context, req *ToolExecutionRequest) (string, error) {
	if req.Tool.Config == nil {
		return "", fmt.Errorf("tool config is required for HTTP tools")
	}

	config := req.Tool.Config
	url := strings.TrimSpace(config.URL)
	if url == "" {
		return "", fmt.Errorf("URL is required for HTTP tools")
	}

	method := strings.ToUpper(strings.TrimSpace(config.Method))
	if method == "" {
		method = "GET"
	}

	// Build payload from template
	var body io.Reader
	if config.PayloadTemplate != "" {
		payload, err := s.renderTemplate(config.PayloadTemplate, req.Arguments)
		if err != nil {
			return "", fmt.Errorf("failed to render payload template: %w", err)
		}
		body = bytes.NewReader([]byte(payload))
		fmt.Println(payload)
	} else if len(req.Arguments) > 0 && (method == "POST" || method == "PUT" || method == "PATCH") {
		// If no template but has arguments, send as JSON
		payloadBytes, err := json.Marshal(req.Arguments)
		if err != nil {
			return "", fmt.Errorf("failed to marshal arguments: %w", err)
		}
		body = bytes.NewReader(payloadBytes)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	for key, value := range config.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add auth headers if configured
	if req.Tool.Auth != nil {
		if apiKey, ok := req.Tool.Auth["api_key"].(string); ok && apiKey != "" {
			if authHeader, ok := req.Tool.Auth["auth_header"].(string); ok && authHeader != "" {
				httpReq.Header.Set(authHeader, apiKey)
			} else {
				httpReq.Header.Set("Authorization", "Bearer "+apiKey)
			}
		}
	}

	// Add query params
	if len(config.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range config.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set timeout
	timeout := defaultToolExecutionTimeout
	if config.TimeoutSeconds != nil && *config.TimeoutSeconds > 0 {
		timeout = time.Duration(*config.TimeoutSeconds) * time.Second
	}

	client := s.client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %s: %s", resp.Status, string(responseBody))
	}

	return string(responseBody), nil
}

// renderTemplate renders a template with arguments
func (s *ToolExecutorService) renderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	// replace {{variable name}} from the data map
	tmpl, err := template.New("payload").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
