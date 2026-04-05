package knowledgebases

import (
	"strings"

	"open-nirmata/db/models"
	"open-nirmata/dto"

	"github.com/gofiber/fiber/v2"
)

var supportedKnowledgebaseProviders = map[string]string{
	"milvus":       string(dto.KnowledgebaseProviderMilvus),
	"mixedbread":   string(dto.KnowledgebaseProviderMixedbread),
	"mixedbreadai": string(dto.KnowledgebaseProviderMixedbread),
	"zeroentropy":  string(dto.KnowledgebaseProviderZeroEntropy),
	"algolia":      string(dto.KnowledgebaseProviderAlgolia),
	"qdrant":       string(dto.KnowledgebaseProviderQdrant),
}

func toKnowledgebaseItem(knowledgebase models.Knowledgebase) dto.KnowledgebaseItem {
	return dto.KnowledgebaseItem{
		Id:             knowledgebase.Id,
		Name:           knowledgebase.Name,
		Provider:       knowledgebase.Provider,
		Description:    knowledgebase.Description,
		Enabled:        knowledgebase.Enabled,
		BaseURL:        knowledgebase.BaseURL,
		IndexName:      knowledgebase.IndexName,
		Namespace:      knowledgebase.Namespace,
		EmbeddingModel: knowledgebase.EmbeddingModel,
		Config:         normalizeLooseMap(knowledgebase.Config),
		AuthConfigured: len(knowledgebase.Auth) > 0,
		CreatedAt:      knowledgebase.CreatedAt,
		UpdatedAt:      knowledgebase.UpdatedAt,
	}
}

func normalizeKnowledgebaseProvider(provider string) (string, bool) {
	normalizedKey := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(provider)))
	normalizedProvider, ok := supportedKnowledgebaseProviders[normalizedKey]
	return normalizedProvider, ok
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

func mergeKnowledgebaseAuth(existing map[string]interface{}, apiKey *string, auth *map[string]interface{}) map[string]interface{} {
	merged := normalizeLooseMap(existing)
	if auth != nil {
		merged = normalizeLooseMap(*auth)
	}

	if apiKey != nil {
		trimmedAPIKey := strings.TrimSpace(*apiKey)
		if trimmedAPIKey == "" {
			if merged != nil {
				delete(merged, "api_key")
			}
		} else {
			if merged == nil {
				merged = map[string]interface{}{}
			}
			merged["api_key"] = trimmedAPIKey
		}
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

func validateKnowledgebaseRecord(knowledgebase models.Knowledgebase) error {
	if strings.TrimSpace(knowledgebase.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	if _, ok := normalizeKnowledgebaseProvider(knowledgebase.Provider); !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid provider; supported values: milvus, mixedbread, zeroentropy, algolia, qdrant")
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
