package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"open-nirmata/dto"
	"open-nirmata/providers"

	"github.com/gofiber/fiber/v2"
)

type fakeMCPService struct {
	result      *dto.TestMCPToolResult
	err         error
	lastConfig  *dto.ToolConfig
	lastTimeout time.Duration
}

func (f *fakeMCPService) ListTools(ctx context.Context, config *dto.ToolConfig, timeout time.Duration) (*dto.TestMCPToolResult, error) {
	f.lastConfig = config
	f.lastTimeout = timeout
	return f.result, f.err
}

func TestTestMCPToolHandlerSuccess(t *testing.T) {
	service := &fakeMCPService{result: &dto.TestMCPToolResult{
		Transport: "stdio",
		Tools: []dto.MCPDiscoveredTool{{
			Name:        "filesystem",
			Description: "Lists files",
		}},
		Count: 1,
	}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{MCP: service}}))
	app.Post("/tools/test", TestMCPTool)

	request := httptest.NewRequest(http.MethodPost, "/tools/test", strings.NewReader(`{"config":{"transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"]},"timeout_seconds":5}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}

	payload := dto.TestMCPToolResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil || payload.Data.Count != 1 {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if service.lastTimeout != 5*time.Second {
		t.Fatalf("expected timeout to be passed through, got %v", service.lastTimeout)
	}
}

func TestTestMCPToolHandlerValidation(t *testing.T) {
	service := &fakeMCPService{err: errors.New("should not be called")}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{MCP: service}}))
	app.Post("/tools/test", TestMCPTool)

	request := httptest.NewRequest(http.MethodPost, "/tools/test", strings.NewReader(`{"config":{"transport":"stdio"}}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to return validation error, got: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", response.StatusCode)
	}
	if service.lastConfig != nil {
		t.Fatalf("expected service not to be called on invalid input")
	}
}
