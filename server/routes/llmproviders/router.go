package llmproviders

import (
	"net/http"

	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

var routeTags = []string{"llm-providers"}

func RegisterRoutes(router fiber.Router, path string) {
	router.Get(path, ListLLMProviders)
	router.Post(path, CreateLLMProvider)
	router.Get(path+"/:id", GetLLMProvider)
	router.Put(path+"/:id", UpdateLLMProvider)
	router.Delete(path+"/:id", DeleteLLMProvider)

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "List LLM Providers",
		Description: "List configured LLM providers with optional provider, enabled, and search filters.",
		Response:    &docs.ApiResponse{Description: "LLM providers fetched successfully", Content: new(dto.LLMProviderListResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "provider", In: "query", Description: "Filter by provider (openai, ollama, anthropic, groq, openrouter, gemini)"},
			{Name: "enabled", In: "query", Description: "Filter by enabled status"},
			{Name: "q", In: "query", Description: "Search by name, description, or default model"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path,
		Method:          http.MethodPost,
		Name:            "Create LLM Provider",
		Description:     "Create a saved LLM provider configuration for OpenAI, Ollama, Anthropic, Groq, OpenRouter, or Gemini.",
		RequestBody:     &docs.ApiRequestBody{Description: "LLM provider details", Content: new(dto.CreateLLMProviderRequest)},
		Response:        &docs.ApiResponse{Description: "LLM provider created successfully", Content: new(dto.LLMProviderResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodGet,
		Name:            "Get LLM Provider",
		Description:     "Fetch a single LLM provider configuration by ID.",
		Response:        &docs.ApiResponse{Description: "LLM provider fetched successfully", Content: new(dto.LLMProviderResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "LLM provider ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodPut,
		Name:            "Update LLM Provider",
		Description:     "Update a saved LLM provider configuration by ID.",
		RequestBody:     &docs.ApiRequestBody{Description: "LLM provider fields to update", Content: new(dto.UpdateLLMProviderRequest)},
		Response:        &docs.ApiResponse{Description: "LLM provider updated successfully", Content: new(dto.LLMProviderResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "LLM provider ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodDelete,
		Name:            "Delete LLM Provider",
		Description:     "Delete a saved LLM provider configuration by ID.",
		Response:        &docs.ApiResponse{Description: "LLM provider deleted successfully", Content: new(dto.LLMProviderResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "LLM provider ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})
}
