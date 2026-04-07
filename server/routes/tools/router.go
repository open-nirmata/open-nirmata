package tools

import (
	"net/http"

	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

var routeTags = []string{"tools"}

func RegisterRoutes(router fiber.Router, path string) {
	router.Get(path, ListTools)
	router.Post(path, CreateTool)
	router.Post(path+"/test", TestMCPTool)
	router.Get(path+"/:id", GetTool)
	router.Put(path+"/:id", UpdateTool)
	router.Post(path+"/:id/refresh", RefreshTool)
	router.Delete(path+"/:id", DeleteTool)

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "List Tools",
		Description: "List configured tools with optional type, provider, enabled, and search filters.",
		Response:    &docs.ApiResponse{Description: "Tools fetched successfully", Content: new(dto.ToolListResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "type", In: "query", Description: "Filter by tool type (mcp, http, llm)"},
			{Name: "provider", In: "query", Description: "Filter LLM tools by provider (openai, ollama, anthropic, groq, openrouter, gemini)"},
			{Name: "enabled", In: "query", Description: "Filter by enabled status"},
			{Name: "q", In: "query", Description: "Search by name or description"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path,
		Method:          http.MethodPost,
		Name:            "Create Tool",
		Description:     "Create a new tool definition for the agent builder. HTTP tools require `config.url` and `config.method`, with optional payload/header/query settings. MCP tools support `transport=stdio` with `command`, `args`, `env`, or `transport=remote` with `server_url` and optional headers/auth.",
		RequestBody:     &docs.ApiRequestBody{Description: "Tool details", Content: new(dto.CreateToolRequest)},
		Response:        &docs.ApiResponse{Description: "Tool created successfully", Content: new(dto.ToolResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/test",
		Method:          http.MethodPost,
		Name:            "Test MCP Tool",
		Description:     "Connect to a stdio or remote MCP server, run the MCP initialize handshake, and return the tools exposed by that server. Supports `npx ...` commands for stdio and remote HTTP/SSE endpoints.",
		RequestBody:     &docs.ApiRequestBody{Description: "Runtime MCP connection settings", Content: new(dto.TestMCPToolRequest)},
		Response:        &docs.ApiResponse{Description: "MCP tools fetched successfully", Content: new(dto.TestMCPToolResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodGet,
		Name:            "Get Tool",
		Description:     "Fetch a single tool definition by ID.",
		Response:        &docs.ApiResponse{Description: "Tool fetched successfully", Content: new(dto.ToolResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Tool ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodPut,
		Name:            "Update Tool",
		Description:     "Update a tool definition by ID. HTTP tools require `config.url` and `config.method`; MCP tools accept stdio (`command`, `args`, `env`) or remote (`server_url`, optional headers/auth) configuration.",
		RequestBody:     &docs.ApiRequestBody{Description: "Tool fields to update", Content: new(dto.UpdateToolRequest)},
		Response:        &docs.ApiResponse{Description: "Tool updated successfully", Content: new(dto.ToolResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Tool ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id/refresh",
		Method:          http.MethodPost,
		Name:            "Refresh MCP Tool",
		Description:     "Reconnect to the configured MCP server for this tool, fetch the latest schema and metadata for the matching tool name, and persist the refreshed details on the existing tool record.",
		RequestBody:     &docs.ApiRequestBody{Description: "Optional timeout override for the refresh operation", Content: new(dto.RefreshToolRequest)},
		Response:        &docs.ApiResponse{Description: "MCP tool refreshed successfully", Content: new(dto.ToolResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Tool ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodDelete,
		Name:            "Delete Tool",
		Description:     "Delete a tool definition by ID.",
		Response:        &docs.ApiResponse{Description: "Tool deleted successfully", Content: new(dto.ToolResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Tool ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})
}
