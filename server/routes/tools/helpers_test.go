package tools

import (
	"testing"

	"open-nirmata/db/models"
	"open-nirmata/dto"
)

func TestNormalizeToolType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{name: "mcp", input: "MCP", expected: "mcp", ok: true},
		{name: "http alias", input: "openapi", expected: "http", ok: true},
		{name: "llm", input: "llm", expected: "llm", ok: true},
		{name: "custom rejected", input: "custom", expected: "", ok: false},
		{name: "other rejected", input: "other", expected: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolType, ok := normalizeToolType(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if toolType != tt.expected {
				t.Fatalf("expected tool type %q, got %q", tt.expected, toolType)
			}
		})
	}
}

func TestValidateToolRecordHTTP(t *testing.T) {
	valid := models.Tool{
		Name: "status-check",
		Type: string(dto.ToolTypeHTTP),
		Config: &dto.ToolConfig{
			URL:             "https://example.com/api",
			Method:          "POST",
			PayloadTemplate: "{\"query\": {{input}}}",
		},
	}

	if err := validateToolRecord(valid); err != nil {
		t.Fatalf("expected valid http tool, got error: %v", err)
	}

	missingURL := valid
	missingURL.Config = &dto.ToolConfig{Method: "GET"}
	if err := validateToolRecord(missingURL); err == nil {
		t.Fatalf("expected error for missing url")
	}

	missingMethod := valid
	missingMethod.Config = &dto.ToolConfig{URL: "https://example.com/api"}
	if err := validateToolRecord(missingMethod); err == nil {
		t.Fatalf("expected error for missing method")
	}
}

func TestValidateToolRecordMCP(t *testing.T) {
	stdioTool := models.Tool{
		Name: "filesystem",
		Type: string(dto.ToolTypeMCP),
		Config: &dto.ToolConfig{
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
		},
	}
	if err := validateToolRecord(stdioTool); err != nil {
		t.Fatalf("expected valid stdio mcp tool, got error: %v", err)
	}

	remoteTool := models.Tool{
		Name: "remote-mcp",
		Type: string(dto.ToolTypeMCP),
		Config: &dto.ToolConfig{
			Transport: "remote",
			ServerURL: "https://mcp.example.com",
		},
	}
	if err := validateToolRecord(remoteTool); err != nil {
		t.Fatalf("expected valid remote mcp tool, got error: %v", err)
	}

	missingCommand := stdioTool
	missingCommand.Config = &dto.ToolConfig{Transport: "stdio"}
	if err := validateToolRecord(missingCommand); err == nil {
		t.Fatalf("expected error for missing stdio command")
	}

	missingServerURL := remoteTool
	missingServerURL.Config = &dto.ToolConfig{Transport: "remote"}
	if err := validateToolRecord(missingServerURL); err == nil {
		t.Fatalf("expected error for missing remote server_url")
	}
}
