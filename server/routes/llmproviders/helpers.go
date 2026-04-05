package llmproviders

import (
	"strings"

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

func validateLLMProviderRecord(provider models.LLMProvider) error {
	if strings.TrimSpace(provider.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	normalizedProvider, ok := normalizeLLMProvider(provider.Provider)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
	}

	if normalizedProvider == string(dto.ToolProviderOllama) {
		if strings.TrimSpace(provider.BaseURL) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "base_url is required for ollama providers")
		}
		return nil
	}

	if apiKeyFromAuth(provider.Auth) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "api_key is required for hosted llm providers")
	}
	return nil
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
