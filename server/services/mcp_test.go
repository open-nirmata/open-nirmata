package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"open-nirmata/dto"
)

func TestMCPServiceListToolsStdio(t *testing.T) {
	service := NewMCPService()
	result, err := service.ListTools(context.Background(), &dto.ToolConfig{
		Transport: "stdio",
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestHelperMCPProcess", "--"},
		Env:       map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("expected stdio MCP call to succeed, got error: %v", err)
	}
	if result == nil || result.Count != 1 {
		t.Fatalf("expected a single discovered tool, got %#v", result)
	}
	if result.ServerInfo == nil || result.ServerInfo.Name != "helper-stdio" {
		t.Fatalf("expected server info to be populated, got %#v", result.ServerInfo)
	}
	if result.Tools[0].Name != "filesystem" {
		t.Fatalf("expected filesystem tool, got %#v", result.Tools[0])
	}
}

func TestMCPServiceListToolsRemoteHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		request := mcpRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		switch request.Method {
		case "initialize":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Mcp-Session-Id", "session-123")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"protocolVersion": defaultMCPProtocolVersion,
					"serverInfo": map[string]interface{}{
						"name":    "remote-http",
						"version": "1.0.0",
					},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			if got := r.Header.Get("Mcp-Session-Id"); got != "session-123" {
				t.Fatalf("expected session header to be forwarded, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "remote-http-tool",
							"description": "Lists remote HTTP tools",
							"inputSchema": map[string]interface{}{"type": "object"},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected remote HTTP method: %s", request.Method)
		}
	}))
	defer server.Close()

	service := NewMCPService()
	result, err := service.ListTools(context.Background(), &dto.ToolConfig{
		Transport: "remote",
		ServerURL: server.URL,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("expected remote HTTP MCP call to succeed, got error: %v", err)
	}
	if result == nil || result.Count != 1 {
		t.Fatalf("expected a single discovered tool, got %#v", result)
	}
	if result.Tools[0].Name != "remote-http-tool" {
		t.Fatalf("expected remote-http-tool, got %#v", result.Tools[0])
	}
}

func TestMCPServiceListToolsRemoteSSE(t *testing.T) {
	responses := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatalf("expected streaming response support")
			}
			fmt.Fprint(w, "event: endpoint\ndata: /messages?sessionId=sse-session\n\n")
			flusher.Flush()
			for {
				select {
				case payload := <-responses:
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", payload)
					flusher.Flush()
				case <-r.Context().Done():
					return
				}
			}
		case "/messages":
			defer r.Body.Close()
			request := mcpRequest{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("failed to decode SSE request: %v", err)
			}

			switch request.Method {
			case "initialize":
				payload, _ := json.Marshal(map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"result": map[string]interface{}{
						"protocolVersion": defaultMCPProtocolVersion,
						"serverInfo": map[string]interface{}{
							"name":    "remote-sse",
							"version": "1.0.0",
						},
					},
				})
				responses <- string(payload)
				w.WriteHeader(http.StatusAccepted)
			case "notifications/initialized":
				w.WriteHeader(http.StatusAccepted)
			case "tools/list":
				payload, _ := json.Marshal(map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"result": map[string]interface{}{
						"tools": []map[string]interface{}{
							{
								"name":        "remote-sse-tool",
								"description": "Lists remote SSE tools",
								"inputSchema": map[string]interface{}{"type": "object"},
							},
						},
					},
				})
				responses <- string(payload)
				w.WriteHeader(http.StatusAccepted)
			default:
				t.Fatalf("unexpected SSE method: %s", request.Method)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewMCPService()
	result, err := service.ListTools(context.Background(), &dto.ToolConfig{
		Transport: "remote",
		ServerURL: server.URL + "/sse",
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("expected remote SSE MCP call to succeed, got error: %v", err)
	}
	if result == nil || result.Count != 1 {
		t.Fatalf("expected a single discovered tool, got %#v", result)
	}
	if result.Tools[0].Name != "remote-sse-tool" {
		t.Fatalf("expected remote-sse-tool, got %#v", result.Tools[0])
	}
}

func TestHelperMCPProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	for {
		payload, err := readFramedJSON(reader)
		if err != nil {
			os.Exit(0)
		}

		request := mcpRequest{}
		if err := json.Unmarshal(payload, &request); err != nil {
			continue
		}

		switch request.Method {
		case "initialize":
			_ = writeFramedJSON(writer, map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"protocolVersion": defaultMCPProtocolVersion,
					"serverInfo": map[string]interface{}{
						"name":    "helper-stdio",
						"version": "1.0.0",
					},
				},
			})
			_ = writer.Flush()
		case "notifications/initialized":
			continue
		case "tools/list":
			_ = writeFramedJSON(writer, map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "filesystem",
							"description": "Test filesystem tool",
							"inputSchema": map[string]interface{}{"type": "object"},
						},
					},
				},
			})
			_ = writer.Flush()
			os.Exit(0)
		}
	}
}
