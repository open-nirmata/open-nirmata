package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"open-nirmata/dto"
)

const defaultMCPProtocolVersion = mcp.LATEST_PROTOCOL_VERSION

type MCPService struct{}

type mcpRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

func NewMCPService() *MCPService {
	return &MCPService{}
}

func (s *MCPService) ListTools(ctx context.Context, config *dto.ToolConfig, timeout time.Duration) (*dto.TestMCPToolResult, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	normalized := normalizeMCPToolConfig(config)
	transport := resolveTransport(normalized)

	switch transport {
	case "stdio":
		client, err := mcpclient.NewStdioMCPClient(normalized.Command, stringMapToEnv(normalized.Env), normalized.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio mcp client: %w", err)
		}
		return s.listToolsWithClient(timedCtx, client, transport, "")
	case "remote":
		return s.listToolsViaRemote(timedCtx, normalized, timeout)
	default:
		return nil, fmt.Errorf("unsupported mcp transport %q", transport)
	}
}

func (s *MCPService) listToolsViaRemote(ctx context.Context, config *dto.ToolConfig, timeout time.Duration) (*dto.TestMCPToolResult, error) {
	serverURL := strings.TrimSpace(config.ServerURL)
	if serverURL == "" {
		return nil, fmt.Errorf("server_url is required for remote mcp tools")
	}

	if looksLikeSSEEndpoint(serverURL) {
		client, err := newSSEClient(serverURL, config.Headers, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create sse mcp client: %w", err)
		}
		return s.listToolsWithClient(ctx, client, "remote", serverURL)
	}

	httpClient, err := newStreamableHTTPClient(serverURL, config.Headers, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable http mcp client: %w", err)
	}

	result, err := s.listToolsWithClient(ctx, httpClient, "remote", serverURL)
	if err == nil {
		return result, nil
	}

	sseClient, sseCreateErr := newSSEClient(serverURL, config.Headers, timeout)
	if sseCreateErr != nil {
		return nil, err
	}
	fallbackResult, fallbackErr := s.listToolsWithClient(ctx, sseClient, "remote", serverURL)
	if fallbackErr == nil {
		return fallbackResult, nil
	}

	if errors.Is(err, mcptransport.ErrLegacySSEServer) {
		return nil, fmt.Errorf("failed to list remote mcp tools via sse fallback: %w", fallbackErr)
	}
	return nil, fmt.Errorf("failed to list remote mcp tools via streamable HTTP (%v) or SSE (%v)", err, fallbackErr)
}

func (s *MCPService) listToolsWithClient(ctx context.Context, client *mcpclient.Client, transport, serverURL string) (*dto.TestMCPToolResult, error) {
	defer client.Close()

	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start mcp client: %w", err)
	}

	initResult, err := client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: defaultMCPProtocolVersion,
			ClientInfo: mcp.Implementation{
				Name:    "open-nirmata",
				Version: "latest",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mcp session: %w", err)
	}

	toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list mcp tools: %w", err)
	}

	discoveredTools := toDiscoveredTools(toolsResult.Tools)
	return &dto.TestMCPToolResult{
		Transport:  transport,
		ServerURL:  strings.TrimSpace(serverURL),
		ServerInfo: toServerInfo(initResult.ServerInfo),
		Tools:      discoveredTools,
		Count:      len(discoveredTools),
	}, nil
}

func newStreamableHTTPClient(serverURL string, headers map[string]string, timeout time.Duration) (*mcpclient.Client, error) {
	options := []mcptransport.StreamableHTTPCOption{
		mcptransport.WithHTTPTimeout(timeout),
	}
	if len(headers) > 0 {
		options = append(options, mcptransport.WithHTTPHeaders(trimStringMap(headers)))
	}
	return mcpclient.NewStreamableHttpClient(serverURL, options...)
}

func newSSEClient(serverURL string, headers map[string]string, timeout time.Duration) (*mcpclient.Client, error) {
	options := []mcptransport.ClientOption{
		mcpclient.WithHTTPClient(&http.Client{Timeout: timeout}),
	}
	if len(headers) > 0 {
		options = append(options, mcpclient.WithHeaders(trimStringMap(headers)))
	}
	return mcpclient.NewSSEMCPClient(serverURL, options...)
}

func resolveTransport(config *dto.ToolConfig) string {
	if config == nil {
		return "stdio"
	}
	if config.Transport != "" {
		return config.Transport
	}
	if config.ServerURL != "" {
		return "remote"
	}
	return "stdio"
}

func looksLikeSSEEndpoint(serverURL string) bool {
	lower := strings.ToLower(strings.TrimSpace(serverURL))
	return strings.Contains(lower, "/sse") || strings.HasSuffix(lower, "sse")
}

func toServerInfo(info mcp.Implementation) *dto.MCPServerInfo {
	name := strings.TrimSpace(info.Name)
	version := strings.TrimSpace(info.Version)
	if name == "" && version == "" {
		return nil
	}
	return &dto.MCPServerInfo{Name: name, Version: version}
}

func toDiscoveredTools(tools []mcp.Tool) []dto.MCPDiscoveredTool {
	if len(tools) == 0 {
		return []dto.MCPDiscoveredTool{}
	}

	result := make([]dto.MCPDiscoveredTool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, dto.MCPDiscoveredTool{
			Name:        strings.TrimSpace(tool.Name),
			Description: strings.TrimSpace(tool.Description),
			InputSchema: extractInputSchema(tool),
			Annotations: toLooseMap(tool.Annotations),
		})
	}
	return result
}

func extractInputSchema(tool mcp.Tool) map[string]interface{} {
	if len(tool.RawInputSchema) > 0 {
		return rawJSONToMap(tool.RawInputSchema)
	}
	return toLooseMap(tool.InputSchema)
}

func rawJSONToMap(payload json.RawMessage) map[string]interface{} {
	if len(payload) == 0 {
		return nil
	}
	decoded := map[string]interface{}{}
	if err := json.Unmarshal(payload, &decoded); err != nil || len(decoded) == 0 {
		return nil
	}
	return decoded
}

func toLooseMap(value interface{}) map[string]interface{} {
	payload, err := json.Marshal(value)
	if err != nil || len(payload) == 0 || string(payload) == "null" {
		return nil
	}
	decoded := map[string]interface{}{}
	if err := json.Unmarshal(payload, &decoded); err != nil || len(decoded) == 0 {
		return nil
	}
	return decoded
}

func writeFramedJSON(writer io.Writer, payload interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(message)); err != nil {
		return err
	}
	_, err = writer.Write(message)
	return err
}

func readFramedJSON(reader *bufio.Reader) (json.RawMessage, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			parsedLength, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length header: %w", err)
			}
			contentLength = parsedLength
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func normalizeMCPToolConfig(config *dto.ToolConfig) *dto.ToolConfig {
	if config == nil {
		return nil
	}

	normalized := *config
	normalized.Transport = strings.ToLower(strings.TrimSpace(config.Transport))
	normalized.Command = strings.TrimSpace(config.Command)
	normalized.Args = normalizeStringList(config.Args)
	normalized.Env = trimStringMap(config.Env)
	normalized.Headers = trimStringMap(config.Headers)
	normalized.ServerURL = strings.TrimSpace(config.ServerURL)
	return &normalized
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func trimStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		result[trimmedKey] = trimmedValue
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func stringMapToEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result = append(result, fmt.Sprintf("%s=%s", trimmedKey, value))
	}
	return result
}
