package tools

import (
	"strings"
	"time"

	"open-nirmata/db/models"
	"open-nirmata/dto"

	"github.com/gofiber/fiber/v2"
)

var supportedToolTypes = map[string]string{
	"mcp":     string(dto.ToolTypeMCP),
	"http":    string(dto.ToolTypeHTTP),
	"openapi": string(dto.ToolTypeHTTP),
	"llm":     string(dto.ToolTypeLLM),
}

var supportedLLMProviders = map[string]string{
	"openai":     string(dto.ToolProviderOpenAI),
	"ollama":     string(dto.ToolProviderOllama),
	"anthropic":  string(dto.ToolProviderAnthropic),
	"groq":       string(dto.ToolProviderGroq),
	"openrouter": string(dto.ToolProviderOpenRouter),
	"gemini":     string(dto.ToolProviderGemini),
}

const defaultMCPTestTimeoutSeconds = 15

func toToolItem(tool models.Tool) dto.ToolItem {
	return dto.ToolItem{
		Id:             tool.Id,
		Name:           tool.Name,
		Type:           tool.Type,
		Provider:       tool.Provider,
		Description:    tool.Description,
		Enabled:        tool.Enabled,
		Tags:           tool.Tags,
		Config:         normalizeToolConfig(tool.Config),
		AuthConfigured: len(normalizeLooseMap(tool.Auth)) > 0,
		CreatedAt:      tool.CreatedAt,
		UpdatedAt:      tool.UpdatedAt,
	}
}

func normalizeToolType(toolType string) (string, bool) {
	normalizedType, ok := supportedToolTypes[strings.ToLower(strings.TrimSpace(toolType))]
	return normalizedType, ok
}

func normalizeProvider(toolType, provider string) (string, error) {
	trimmedProvider := strings.TrimSpace(provider)
	if trimmedProvider == "" {
		if toolType == string(dto.ToolTypeLLM) {
			return "", fiber.NewError(fiber.StatusBadRequest, "provider is required for llm tools")
		}
		return "", nil
	}

	normalizedKey := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(trimmedProvider))
	normalizedProvider, ok := supportedLLMProviders[normalizedKey]
	if toolType == string(dto.ToolTypeLLM) {
		if !ok {
			return "", fiber.NewError(fiber.StatusBadRequest, "invalid llm provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
		}
		return normalizedProvider, nil
	}

	if ok {
		return normalizedProvider, nil
	}
	return trimmedProvider, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmedTag := strings.TrimSpace(tag)
		if trimmedTag == "" {
			continue
		}
		key := strings.ToLower(trimmedTag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmedTag)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeLooseMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}

	normalized := make(map[string]interface{}, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = value
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		normalized[trimmedKey] = trimmedValue
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStringList(input []string) []string {
	if len(input) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeToolConfig(config *dto.ToolConfig) *dto.ToolConfig {
	if config == nil {
		return nil
	}

	normalized := &dto.ToolConfig{
		URL:             strings.TrimSpace(config.URL),
		Method:          strings.ToUpper(strings.TrimSpace(config.Method)),
		PayloadTemplate: strings.TrimSpace(config.PayloadTemplate),
		Headers:         normalizeStringMap(config.Headers),
		QueryParams:     normalizeStringMap(config.QueryParams),
		Transport:       strings.ToLower(strings.TrimSpace(config.Transport)),
		Command:         strings.TrimSpace(config.Command),
		Args:            normalizeStringList(config.Args),
		Env:             normalizeStringMap(config.Env),
		ServerURL:       strings.TrimSpace(config.ServerURL),
	}

	if config.TimeoutSeconds != nil {
		timeout := *config.TimeoutSeconds
		normalized.TimeoutSeconds = &timeout
	}

	if normalized.URL == "" && normalized.Method == "" && normalized.PayloadTemplate == "" &&
		len(normalized.Headers) == 0 && len(normalized.QueryParams) == 0 && normalized.TimeoutSeconds == nil &&
		normalized.Transport == "" && normalized.Command == "" && len(normalized.Args) == 0 &&
		len(normalized.Env) == 0 && normalized.ServerURL == "" {
		return nil
	}

	return normalized
}

func validateToolRecord(tool models.Tool) error {
	if strings.TrimSpace(tool.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	normalizedType, ok := normalizeToolType(tool.Type)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tool type; supported values: mcp, http, llm")
	}

	if _, err := normalizeProvider(normalizedType, tool.Provider); err != nil {
		return err
	}

	switch normalizedType {
	case string(dto.ToolTypeHTTP):
		return validateHTTPConfig(tool.Config)
	case string(dto.ToolTypeMCP):
		return validateMCPConfig(tool.Config)
	default:
		return nil
	}
}

func validateHTTPConfig(config *dto.ToolConfig) error {
	normalized := normalizeToolConfig(config)
	if normalized == nil || normalized.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required for http tools")
	}
	if normalized.Method == "" {
		return fiber.NewError(fiber.StatusBadRequest, "method is required for http tools")
	}
	if normalized.TimeoutSeconds != nil && *normalized.TimeoutSeconds <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "timeout_seconds must be a positive number when provided")
	}
	return nil
}

func validateMCPConfig(config *dto.ToolConfig) error {
	normalized := normalizeToolConfig(config)
	transport := "stdio"
	if normalized != nil && normalized.Transport != "" {
		transport = normalized.Transport
	} else if normalized != nil && normalized.ServerURL != "" {
		transport = "remote"
	}

	switch transport {
	case "stdio":
		if normalized == nil || normalized.Command == "" {
			return fiber.NewError(fiber.StatusBadRequest, "command is required for stdio mcp tools")
		}
	case "remote":
		if normalized == nil || normalized.ServerURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "server_url is required for remote mcp tools")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "transport must be either stdio or remote for mcp tools")
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

func badGateway(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadGateway).JSON(dto.ErrorResponse{Success: false, Message: message})
}

func resolveMCPTestTimeout(timeoutSeconds *int) (time.Duration, error) {
	timeout := defaultMCPTestTimeoutSeconds
	if timeoutSeconds != nil {
		if *timeoutSeconds <= 0 {
			return 0, fiber.NewError(fiber.StatusBadRequest, "timeout_seconds must be a positive number when provided")
		}
		timeout = *timeoutSeconds
	}
	return time.Duration(timeout) * time.Second, nil
}
