package promptflows

import (
	"net/http"

	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

var routeTags = []string{"prompt-flows"}

func RegisterRoutes(router fiber.Router, path string) {
	router.Get(path, ListPromptFlows)
	router.Post(path, CreatePromptFlow)
	router.Post(path+"/validate", ValidatePromptFlow)
	router.Get(path+"/:id", GetPromptFlow)
	router.Put(path+"/:id", UpdatePromptFlow)
	router.Delete(path+"/:id", DeletePromptFlow)

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "List Prompt Flows",
		Description: "List prompt flow definitions for chat agents with optional enabled and search filters.",
		Response:    &docs.ApiResponse{Description: "Prompt flows fetched successfully", Content: new(dto.PromptFlowListResponse)},
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
		Name:            "Create Prompt Flow",
		Description:     "Create a prompt flow configuration for chat agents with flow-level defaults and stage-level overrides. This endpoint stores design-time configuration only; it does not execute the flow.",
		RequestBody:     &docs.ApiRequestBody{Description: "Prompt flow details", Content: new(dto.CreatePromptFlowRequest)},
		Response:        &docs.ApiResponse{Description: "Prompt flow created successfully", Content: new(dto.PromptFlowResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/validate",
		Method:          http.MethodPost,
		Name:            "Validate Prompt Flow",
		Description:     "Validate a prompt flow definition, including stage graph integrity and referenced llm providers, tools, and knowledgebases. This endpoint does not execute prompts or tools.",
		RequestBody:     &docs.ApiRequestBody{Description: "Prompt flow to validate", Content: new(dto.ValidatePromptFlowRequest)},
		Response:        &docs.ApiResponse{Description: "Prompt flow validated successfully", Content: new(dto.PromptFlowValidateResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodGet,
		Name:            "Get Prompt Flow",
		Description:     "Fetch a single prompt flow definition by ID.",
		Response:        &docs.ApiResponse{Description: "Prompt flow fetched successfully", Content: new(dto.PromptFlowResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Prompt flow ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodPut,
		Name:            "Update Prompt Flow",
		Description:     "Update a saved prompt flow definition by ID.",
		RequestBody:     &docs.ApiRequestBody{Description: "Prompt flow fields to update", Content: new(dto.UpdatePromptFlowRequest)},
		Response:        &docs.ApiResponse{Description: "Prompt flow updated successfully", Content: new(dto.PromptFlowResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Prompt flow ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodDelete,
		Name:            "Delete Prompt Flow",
		Description:     "Delete a saved prompt flow definition by ID.",
		Response:        &docs.ApiResponse{Description: "Prompt flow deleted successfully", Content: new(dto.PromptFlowResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Prompt flow ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})
}
