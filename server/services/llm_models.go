package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"open-nirmata/dto"
)

var llmModelsSupportedProviders = map[string]string{
	"openai":     string(dto.ToolProviderOpenAI),
	"ollama":     string(dto.ToolProviderOllama),
	"anthropic":  string(dto.ToolProviderAnthropic),
	"groq":       string(dto.ToolProviderGroq),
	"openrouter": string(dto.ToolProviderOpenRouter),
	"gemini":     string(dto.ToolProviderGemini),
}

const defaultLLMModelsTimeout = 15 * time.Second

type LLMModelsService struct {
	client *http.Client
}

func NewLLMModelsService() *LLMModelsService {
	return &LLMModelsService{client: &http.Client{}}
}

func (s *LLMModelsService) ListModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest, timeout time.Duration) ([]dto.LLMModelItem, error) {
	normalizedReq, err := normalizeLLMModelsRequest(req)
	if err != nil {
		return nil, err
	}

	if timeout <= 0 {
		timeout = defaultLLMModelsTimeout
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch normalizedReq.Provider {
	case string(dto.ToolProviderOpenAI):
		headers := map[string]string{"Authorization": "Bearer " + normalizedReq.APIKey}
		if normalizedReq.Organization != "" {
			headers["OpenAI-Organization"] = normalizedReq.Organization
		}
		if normalizedReq.ProjectID != "" {
			headers["OpenAI-Project"] = normalizedReq.ProjectID
		}
		return s.listOpenAIStyleModels(timedCtx, normalizedReq.Provider, normalizedReq.BaseURL, "https://api.openai.com", "/v1/models", headers)
	case string(dto.ToolProviderGroq):
		return s.listOpenAIStyleModels(timedCtx, normalizedReq.Provider, normalizedReq.BaseURL, "https://api.groq.com/openai", "/v1/models", map[string]string{"Authorization": "Bearer " + normalizedReq.APIKey})
	case string(dto.ToolProviderOpenRouter):
		return s.listOpenAIStyleModels(timedCtx, normalizedReq.Provider, normalizedReq.BaseURL, "https://openrouter.ai/api/v1", "/models", map[string]string{"Authorization": "Bearer " + normalizedReq.APIKey})
	case string(dto.ToolProviderAnthropic):
		return s.listAnthropicModels(timedCtx, normalizedReq)
	case string(dto.ToolProviderGemini):
		return s.listGeminiModels(timedCtx, normalizedReq)
	case string(dto.ToolProviderOllama):
		return s.listOllamaModels(timedCtx, normalizedReq)
	default:
		return nil, fmt.Errorf("unsupported provider %q", normalizedReq.Provider)
	}
}

func (s *LLMModelsService) listOpenAIStyleModels(ctx context.Context, provider, baseURL, defaultBaseURL, path string, headers map[string]string) ([]dto.LLMModelItem, error) {
	endpoint := resolveEndpoint(baseURL, defaultBaseURL, path)
	payload := map[string]interface{}{}
	if err := s.fetchJSON(ctx, endpoint, headers, &payload); err != nil {
		return nil, err
	}

	return normalizeOpenAIStyleItems(provider, toInterfaceSlice(payload["data"])), nil
}

func normalizeOpenAIStyleItems(provider string, items []interface{}) []dto.LLMModelItem {
	result := make([]dto.LLMModelItem, 0, len(items))
	for _, rawItem := range items {
		item := toLooseMap(rawItem)
		if len(item) == 0 {
			continue
		}

		id := firstNonEmpty(stringValue(item["id"]), stringValue(item["name"]))
		if id == "" {
			continue
		}

		architecture := toLooseMap(item["architecture"])
		capabilities := uniqueStrings(combineStringSlices(
			stringSlice(item["supported_generation_methods"]),
			stringSlice(item["supportedGenerationMethods"]),
			stringSlice(architecture["input_modalities"]),
			stringSlice(architecture["output_modalities"]),
		)...)

		result = append(result, dto.LLMModelItem{
			ID:               id,
			Name:             firstNonEmpty(stringValue(item["name"]), stringValue(item["display_name"]), stringValue(item["displayName"]), lastPathToken(id), id),
			Provider:         provider,
			Description:      firstNonEmpty(stringValue(item["description"]), stringValue(item["display_name"]), stringValue(item["displayName"])),
			OwnedBy:          stringValue(item["owned_by"]),
			ContextWindow:    firstPositiveInt(intValue(item["context_window"]), intValue(item["context_length"]), intValue(item["contextLength"]), intValue(item["max_context_tokens"])),
			InputTokenLimit:  firstPositiveInt(intValue(item["input_token_limit"]), intValue(item["inputTokenLimit"])),
			OutputTokenLimit: firstPositiveInt(intValue(item["output_token_limit"]), intValue(item["outputTokenLimit"])),
			Capabilities:     capabilities,
			Raw:              item,
		})
	}

	return result
}

func (s *LLMModelsService) listAnthropicModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest) ([]dto.LLMModelItem, error) {
	endpoint := resolveEndpoint(req.BaseURL, "https://api.anthropic.com", "/v1/models")
	payload := map[string]interface{}{}
	if err := s.fetchJSON(ctx, endpoint, map[string]string{
		"x-api-key":         req.APIKey,
		"anthropic-version": "2023-06-01",
	}, &payload); err != nil {
		return nil, err
	}
	return normalizeOpenAIStyleItems(req.Provider, toInterfaceSlice(payload["data"])), nil
}

func (s *LLMModelsService) listGeminiModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest) ([]dto.LLMModelItem, error) {
	endpoint := resolveEndpoint(req.BaseURL, "https://generativelanguage.googleapis.com", "/v1beta/models")
	endpoint = addQueryParam(endpoint, "key", req.APIKey)

	payload := map[string]interface{}{}
	if err := s.fetchJSON(ctx, endpoint, nil, &payload); err != nil {
		return nil, err
	}

	items := toInterfaceSlice(payload["models"])
	result := make([]dto.LLMModelItem, 0, len(items))
	for _, rawItem := range items {
		item := toLooseMap(rawItem)
		if len(item) == 0 {
			continue
		}

		id := stringValue(item["name"])
		if id == "" {
			continue
		}

		inputLimit := firstPositiveInt(intValue(item["inputTokenLimit"]), intValue(item["input_token_limit"]))
		result = append(result, dto.LLMModelItem{
			ID:               id,
			Name:             firstNonEmpty(stringValue(item["displayName"]), stringValue(item["display_name"]), lastPathToken(id), id),
			Provider:         req.Provider,
			Description:      stringValue(item["description"]),
			ContextWindow:    inputLimit,
			InputTokenLimit:  inputLimit,
			OutputTokenLimit: firstPositiveInt(intValue(item["outputTokenLimit"]), intValue(item["output_token_limit"])),
			Capabilities:     uniqueStrings(stringSlice(item["supportedGenerationMethods"])...),
			Raw:              item,
		})
	}

	return result, nil
}

func (s *LLMModelsService) listOllamaModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest) ([]dto.LLMModelItem, error) {
	endpoint := resolveEndpoint(req.BaseURL, req.BaseURL, "/api/tags")
	payload := map[string]interface{}{}
	if err := s.fetchJSON(ctx, endpoint, nil, &payload); err != nil {
		return nil, err
	}

	items := toInterfaceSlice(payload["models"])
	result := make([]dto.LLMModelItem, 0, len(items))
	for _, rawItem := range items {
		item := toLooseMap(rawItem)
		if len(item) == 0 {
			continue
		}

		details := toLooseMap(item["details"])
		id := firstNonEmpty(stringValue(item["model"]), stringValue(item["name"]))
		if id == "" {
			continue
		}

		capabilities := uniqueStrings(combineStringSlices(
			stringSlice(details["families"]),
			stringSlice(details["family"]),
		)...)

		result = append(result, dto.LLMModelItem{
			ID:            id,
			Name:          firstNonEmpty(stringValue(item["name"]), lastPathToken(id), id),
			Provider:      req.Provider,
			Description:   firstNonEmpty(stringValue(item["description"]), stringValue(details["parameter_size"]), stringValue(details["format"])),
			Capabilities:  capabilities,
			ContextWindow: firstPositiveInt(intValue(details["context_length"]), intValue(details["contextLength"])),
			Raw:           item,
		})
	}

	return result, nil
}

func (s *LLMModelsService) fetchJSON(ctx context.Context, endpoint string, headers map[string]string, target interface{}) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create provider request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		request.Header.Set(key, value)
	}

	client := s.client
	if client == nil {
		client = &http.Client{}
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("provider request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := sanitizeErrorBody(body)
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("provider returned %s: %s", response.Status, message)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode provider response: %w", err)
	}
	return nil
}

func normalizeLLMModelsRequest(req *dto.ListLLMProviderModelsRequest) (*dto.ListLLMProviderModelsRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	normalizedProvider, ok := normalizeLLMModelProvider(req.Provider)
	if !ok {
		return nil, fmt.Errorf("invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
	}

	normalized := &dto.ListLLMProviderModelsRequest{
		Provider:     normalizedProvider,
		BaseURL:      strings.TrimSpace(req.BaseURL),
		APIKey:       strings.TrimSpace(req.APIKey),
		Organization: strings.TrimSpace(req.Organization),
		ProjectID:    strings.TrimSpace(req.ProjectID),
	}
	if req.TimeoutSeconds != nil {
		timeout := *req.TimeoutSeconds
		normalized.TimeoutSeconds = &timeout
	}

	if normalized.Provider == string(dto.ToolProviderOllama) {
		if normalized.BaseURL == "" {
			return nil, fmt.Errorf("base_url is required for ollama providers")
		}
		return normalized, nil
	}

	if normalized.APIKey == "" {
		return nil, fmt.Errorf("api_key is required for hosted llm providers")
	}
	return normalized, nil
}

func normalizeLLMModelProvider(provider string) (string, bool) {
	normalizedKey := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(provider)))
	normalizedProvider, ok := llmModelsSupportedProviders[normalizedKey]
	return normalizedProvider, ok
}

func resolveEndpoint(baseURL, defaultBaseURL, path string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(defaultBaseURL), "/")
	}
	trimmedPath := "/" + strings.TrimLeft(path, "/")
	if trimmedPath == "/" || trimmedPath == "" {
		return base
	}
	if strings.HasSuffix(base, trimmedPath) {
		return base
	}
	if suffix := strings.TrimSuffix(trimmedPath, "/models"); suffix != "" && strings.HasSuffix(base, suffix) {
		return base + strings.TrimPrefix(trimmedPath, suffix)
	}
	if suffix := strings.TrimSuffix(trimmedPath, "/tags"); suffix != "" && strings.HasSuffix(base, suffix) {
		return base + strings.TrimPrefix(trimmedPath, suffix)
	}
	return base + trimmedPath
}

func addQueryParam(endpoint, key, value string) string {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsedURL.Query()
	query.Set(key, value)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func sanitizeErrorBody(body []byte) string {
	message := strings.TrimSpace(string(body))
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	return strings.TrimSpace(message)
}

func toInterfaceSlice(value interface{}) []interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	return items
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func stringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	case []string:
		return uniqueStrings(typed...)
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := stringValue(item); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return uniqueStrings(values...)
	default:
		return nil
	}
}

func combineStringSlices(values ...[]string) []string {
	combined := []string{}
	for _, group := range values {
		combined = append(combined, group...)
	}
	return combined
}

func uniqueStrings(values ...string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func lastPathToken(value string) string {
	trimmed := strings.Trim(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}
