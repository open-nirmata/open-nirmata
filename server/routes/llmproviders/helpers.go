package llmproviders

import (
	"strings"
	"time"

	"open-nirmata/db/models"
	"open-nirmata/dto"

	"github.com/gofiber/fiber/v2"
)

var supportedProviders = map[string]string{
	"openai":     string(dto.ToolProviderOpenAI),
	"ollama":     string(dto.ToolProviderOllama),
	"anthropic":  string(dto.ToolProviderAnthropic),
	"groq":       string(dto.ToolProviderGroq),
	"openrouter": string(dto.ToolProviderOpenRouter),
	"gemini":     string(dto.ToolProviderGemini),
}

const defaultLLMModelsTimeoutSeconds = 15

func toLLMProviderItem(provider models.LLMProvider) dto.LLMProviderItem {
	return dto.LLMProviderItem{
		Id:             provider.Id,
		Name:           provider.Name,
		Provider:       provider.Provider,
		Description:    provider.Description,
		Enabled:        provider.Enabled,
		BaseURL:        provider.BaseURL,
		DefaultModel:   provider.DefaultModel,
		Organization:   provider.Organization,
		ProjectID:      provider.ProjectID,
		AuthConfigured: len(provider.Auth) > 0,
		CreatedAt:      provider.CreatedAt,
		UpdatedAt:      provider.UpdatedAt,
	}
}

func normalizeLLMProvider(provider string) (string, bool) {
	normalizedKey := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(provider)))
	normalizedProvider, ok := supportedProviders[normalizedKey]
	return normalizedProvider, ok
}

func buildProviderAuth(apiKey string) map[string]interface{} {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	if trimmedAPIKey == "" {
		return nil
	}
	return map[string]interface{}{"api_key": trimmedAPIKey}
}

func apiKeyFromAuth(auth map[string]interface{}) string {
	if len(auth) == 0 {
		return ""
	}
	value, ok := auth["api_key"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func normalizeLLMProviderModelsRequest(req *dto.ListLLMProviderModelsRequest) *dto.ListLLMProviderModelsRequest {
	if req == nil {
		return nil
	}

	normalized := &dto.ListLLMProviderModelsRequest{
		LLMProviderID: strings.TrimSpace(req.LLMProviderID),
		Provider:      strings.TrimSpace(req.Provider),
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKey:        strings.TrimSpace(req.APIKey),
		Organization:  strings.TrimSpace(req.Organization),
		ProjectID:     strings.TrimSpace(req.ProjectID),
	}
	if req.TimeoutSeconds != nil {
		timeout := *req.TimeoutSeconds
		normalized.TimeoutSeconds = &timeout
	}
	if normalizedProvider, ok := normalizeLLMProvider(normalized.Provider); ok {
		normalized.Provider = normalizedProvider
	}
	return normalized
}

func validateLLMProviderModelsRequest(req *dto.ListLLMProviderModelsRequest) error {
	if req == nil {
		return fiber.NewError(fiber.StatusBadRequest, "request body is required")
	}

	normalizedProvider, ok := normalizeLLMProvider(req.Provider)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
	}
	req.Provider = normalizedProvider

	if normalizedProvider == string(dto.ToolProviderOllama) {
		if strings.TrimSpace(req.BaseURL) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "base_url is required for ollama providers")
		}
		return nil
	}

	if strings.TrimSpace(req.APIKey) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "api_key is required for hosted llm providers")
	}
	return nil
}

func mergeLLMProviderModelsRequest(savedProvider models.LLMProvider, req *dto.ListLLMProviderModelsRequest) *dto.ListLLMProviderModelsRequest {
	if req == nil {
		req = &dto.ListLLMProviderModelsRequest{}
	}

	merged := &dto.ListLLMProviderModelsRequest{
		LLMProviderID: strings.TrimSpace(req.LLMProviderID),
		Provider:      strings.TrimSpace(req.Provider),
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKey:        strings.TrimSpace(req.APIKey),
		Organization:  strings.TrimSpace(req.Organization),
		ProjectID:     strings.TrimSpace(req.ProjectID),
	}
	if req.TimeoutSeconds != nil {
		timeout := *req.TimeoutSeconds
		merged.TimeoutSeconds = &timeout
	}

	if merged.Provider == "" {
		merged.Provider = strings.TrimSpace(savedProvider.Provider)
	}
	if merged.BaseURL == "" {
		merged.BaseURL = strings.TrimSpace(savedProvider.BaseURL)
	}
	if merged.APIKey == "" {
		merged.APIKey = apiKeyFromAuth(savedProvider.Auth)
	}
	if merged.Organization == "" {
		merged.Organization = strings.TrimSpace(savedProvider.Organization)
	}
	if merged.ProjectID == "" {
		merged.ProjectID = strings.TrimSpace(savedProvider.ProjectID)
	}
	if normalizedProvider, ok := normalizeLLMProvider(merged.Provider); ok {
		merged.Provider = normalizedProvider
	}
	return merged
}

func validateLLMProviderRecord(provider models.LLMProvider) error {
	if strings.TrimSpace(provider.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	return validateLLMProviderModelsRequest(&dto.ListLLMProviderModelsRequest{
		Provider: provider.Provider,
		BaseURL:  provider.BaseURL,
		APIKey:   apiKeyFromAuth(provider.Auth),
	})
}

func resolveLLMModelsTimeout(timeoutSeconds *int) (time.Duration, error) {
	timeout := defaultLLMModelsTimeoutSeconds
	if timeoutSeconds != nil {
		if *timeoutSeconds <= 0 {
			return 0, fiber.NewError(fiber.StatusBadRequest, "timeout_seconds must be a positive number when provided")
		}
		timeout = *timeoutSeconds
	}
	return time.Duration(timeout) * time.Second, nil
}

func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Success: false, Message: message})
}

func notFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Success: false, Message: message})
}

func internalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Success: false, Message: message})
}

func badGateway(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResponse{Success: false, Message: message})
}
