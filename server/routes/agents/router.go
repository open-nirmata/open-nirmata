package agents

import (
	"net/http"

	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

var routeTags = []string{"agents"}

func RegisterRoutes(router fiber.Router, path string) {
	router.Get(path, ListAgents)
	router.Post(path, CreateAgent)
	router.Post(path+"/validate", ValidateAgent)
	router.Get(path+"/:id", GetAgent)
	router.Put(path+"/:id", UpdateAgent)
	router.Delete(path+"/:id", DeleteAgent)
	router.Post(path+"/:id/execute", ExecuteAgent)

	// Execution history routes
	router.Get("/executions", ListExecutions)
	router.Get("/executions/:id", GetExecution)

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "List Agents",
		Description: "List chat agents with optional enabled and search filters.",
		Response:    &docs.ApiResponse{Description: "Agents fetched successfully", Content: new(dto.AgentListResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "enabled", In: "query", Description: "Filter by enabled status"},
			{Name: "q", In: "query", Description: "Search by name or description"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path,
		Method:          http.MethodPost,
		Name:            "Create Agent",
		Description:     "Create a chat agent that references a prompt flow. This endpoint stores the agent configuration only; agent execution APIs are implemented separately.",
		RequestBody:     &docs.ApiRequestBody{Description: "Agent details", Content: new(dto.CreateAgentRequest)},
		Response:        &docs.ApiResponse{Description: "Agent created successfully", Content: new(dto.AgentResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/validate",
		Method:          http.MethodPost,
		Name:            "Validate Agent",
		Description:     "Validate an agent definition, including checking required fields, agent type, and prompt flow reference. This endpoint does not create or modify the database.",
		RequestBody:     &docs.ApiRequestBody{Description: "Agent to validate", Content: new(dto.ValidateAgentRequest)},
		Response:        &docs.ApiResponse{Description: "Agent validated successfully", Content: new(dto.AgentResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodGet,
		Name:            "Get Agent",
		Description:     "Fetch a single agent by ID.",
		Response:        &docs.ApiResponse{Description: "Agent fetched successfully", Content: new(dto.AgentResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Agent ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodPut,
		Name:            "Update Agent",
		Description:     "Update a saved agent by ID.",
		RequestBody:     &docs.ApiRequestBody{Description: "Agent fields to update", Content: new(dto.UpdateAgentRequest)},
		Response:        &docs.ApiResponse{Description: "Agent updated successfully", Content: new(dto.AgentResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Agent ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodDelete,
		Name:            "Delete Agent",
		Description:     "Delete a saved agent by ID.",
		Response:        &docs.ApiResponse{Description: "Agent deleted successfully", Content: new(dto.AgentResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Agent ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path + "/:id/execute",
		Method:      http.MethodPost,
		Name:        "Execute Agent",
		Description: "Execute an agent using its prompt flow. Supports three modes: synchronous (default), asynchronous (?async=true), and server-sent events streaming (?stream=true). The execution traverses the prompt flow stages (chat, tool, retrieval, router) and records comprehensive execution history including LLM interactions, tool calls, and stage transitions.",
		RequestBody: &docs.ApiRequestBody{Description: "Execution request with user message(s)", Content: new(dto.ExecuteAgentRequest)},
		Response:    &docs.ApiResponse{Description: "Execution result (data for sync, job_id for async, SSE events for streaming)", Content: new(dto.ExecuteAgentResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "id", In: "path", Description: "Agent ID", Required: true},
			{Name: "async", In: "query", Description: "Execute asynchronously and return job ID"},
			{Name: "stream", In: "query", Description: "Stream execution events via Server-Sent Events"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:        "/executions",
		Method:      http.MethodGet,
		Name:        "List Executions",
		Description: "List recent execution history records. Supports filtering by agent and status, and limiting the number of results returned.",
		Response:    &docs.ApiResponse{Description: "Executions fetched successfully", Content: new(dto.ListExecutionsResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "agent_id", In: "query", Description: "Filter by agent ID"},
			{Name: "status", In: "query", Description: "Filter by execution status"},
			{Name: "limit", In: "query", Description: "Limit the number of execution records returned (max 100)"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            "/executions/:id",
		Method:          http.MethodGet,
		Name:            "Get Execution",
		Description:     "Retrieve execution history by execution ID. Returns the complete execution record including all steps, LLM interactions, tool calls, and metadata.",
		Response:        &docs.ApiResponse{Description: "Execution fetched successfully", Content: new(dto.ExecutionItem)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Execution ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})
}
