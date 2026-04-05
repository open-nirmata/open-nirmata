package llmproviders

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"open-nirmata/db/models"
	"open-nirmata/dto"
	"open-nirmata/providers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ListLLMProviders(c *fiber.Ctx) error {
	database := providers.GetProviders(c).D
	providerModel := models.GetLLMProviderModel()
	filter := bson.M{}

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		normalizedProvider, ok := normalizeLLMProvider(provider)
		if !ok {
			return badRequest(c, "invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
		}
		filter[providerModel.ProviderKey] = normalizedProvider
	}

	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		parsedEnabled, err := strconv.ParseBool(enabled)
		if err != nil {
			return badRequest(c, "invalid enabled value")
		}
		filter[providerModel.EnabledKey] = parsedEnabled
	}

	if searchText := strings.TrimSpace(c.Query("q")); searchText != "" {
		filter["$or"] = bson.A{
			bson.M{providerModel.NameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{providerModel.DescriptionKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{providerModel.DefaultModelKey: bson.M{"$regex": searchText, "$options": "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: providerModel.CreatedAtKey, Value: -1}})
	cursor, err := database.Find(c.Context(), providerModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list llm providers")
	}
	defer cursor.Close(c.Context())

	items := make([]models.LLMProvider, 0)
	if err := cursor.All(c.Context(), &items); err != nil {
		return internalError(c, "failed to decode llm providers")
	}

	response := make([]dto.LLMProviderItem, 0, len(items))
	for _, item := range items {
		response = append(response, toLLMProviderItem(item))
	}

	return c.JSON(dto.LLMProviderListResponse{Success: true, Data: response, Count: len(response)})
}

func GetLLMProvider(c *fiber.Ctx) error {
	item, err := findLLMProviderByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "llm provider not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load llm provider")
	}

	return c.JSON(dto.LLMProviderResponse{Success: true, Data: &item})
}

func CreateLLMProvider(c *fiber.Ctx) error {
	var req dto.CreateLLMProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return badRequest(c, "name is required")
	}

	normalizedProvider, ok := normalizeLLMProvider(req.Provider)
	if !ok {
		return badRequest(c, "invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC()
	provider := models.LLMProvider{
		Id:           uuid.NewString(),
		Name:         name,
		Provider:     normalizedProvider,
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		BaseURL:      strings.TrimSpace(req.BaseURL),
		DefaultModel: strings.TrimSpace(req.DefaultModel),
		Organization: strings.TrimSpace(req.Organization),
		ProjectID:    strings.TrimSpace(req.ProjectID),
		Auth:         buildProviderAuth(req.APIKey),
		CreatedAt:    &now,
		CreatedBy:    "system",
		UpdatedAt:    &now,
		UpdatedBy:    "system",
	}

	if err := validateLLMProviderRecord(provider); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate llm provider")
	}

	database := providers.GetProviders(c).D
	providerModel := models.GetLLMProviderModel()
	count, err := database.CountDocuments(c.Context(), providerModel, bson.M{
		providerModel.NameKey:     provider.Name,
		providerModel.ProviderKey: provider.Provider,
	})
	if err != nil {
		return internalError(c, "failed to validate llm provider uniqueness")
	}
	if count > 0 {
		return badRequest(c, "llm provider with the same name and provider already exists")
	}

	if _, err := database.InsertOne(c.Context(), providerModel, provider); err != nil {
		return internalError(c, "failed to create llm provider")
	}

	item := toLLMProviderItem(provider)
	return c.Status(fiber.StatusCreated).JSON(dto.LLMProviderResponse{Success: true, Data: &item, Message: "llm provider created successfully"})
}

func UpdateLLMProvider(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	var req dto.UpdateLLMProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	existing, err := loadLLMProviderByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "llm provider not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load llm provider")
	}

	updated := existing
	changed := false

	if req.Name != nil {
		updated.Name = strings.TrimSpace(*req.Name)
		if updated.Name == "" {
			return badRequest(c, "name cannot be empty")
		}
		changed = true
	}
	if req.Provider != nil {
		normalizedProvider, ok := normalizeLLMProvider(*req.Provider)
		if !ok {
			return badRequest(c, "invalid provider; supported values: openai, ollama, anthropic, groq, openrouter, gemini")
		}
		updated.Provider = normalizedProvider
		changed = true
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
		changed = true
	}
	if req.Enabled != nil {
		updated.Enabled = *req.Enabled
		changed = true
	}
	if req.BaseURL != nil {
		updated.BaseURL = strings.TrimSpace(*req.BaseURL)
		changed = true
	}
	if req.DefaultModel != nil {
		updated.DefaultModel = strings.TrimSpace(*req.DefaultModel)
		changed = true
	}
	if req.Organization != nil {
		updated.Organization = strings.TrimSpace(*req.Organization)
		changed = true
	}
	if req.ProjectID != nil {
		updated.ProjectID = strings.TrimSpace(*req.ProjectID)
		changed = true
	}
	if req.APIKey != nil {
		updated.Auth = buildProviderAuth(*req.APIKey)
		changed = true
	}

	if !changed {
		return badRequest(c, "no fields to update")
	}

	if err := validateLLMProviderRecord(updated); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate llm provider")
	}

	database := providers.GetProviders(c).D
	providerModel := models.GetLLMProviderModel()
	count, err := database.CountDocuments(c.Context(), providerModel, bson.M{
		providerModel.NameKey:     updated.Name,
		providerModel.ProviderKey: updated.Provider,
		providerModel.IdKey:       bson.M{"$ne": id},
	})
	if err != nil {
		return internalError(c, "failed to validate llm provider uniqueness")
	}
	if count > 0 {
		return badRequest(c, "llm provider with the same name and provider already exists")
	}

	now := time.Now().UTC()
	updateFields := bson.M{
		providerModel.NameKey:         updated.Name,
		providerModel.ProviderKey:     updated.Provider,
		providerModel.DescriptionKey:  updated.Description,
		providerModel.EnabledKey:      updated.Enabled,
		providerModel.BaseURLKey:      updated.BaseURL,
		providerModel.DefaultModelKey: updated.DefaultModel,
		providerModel.OrganizationKey: updated.Organization,
		providerModel.ProjectIDKey:    updated.ProjectID,
		providerModel.AuthKey:         updated.Auth,
		providerModel.UpdatedAtKey:    &now,
		providerModel.UpdatedByKey:    "system",
	}

	result, err := database.UpdateOne(c.Context(), providerModel, bson.M{providerModel.IdKey: id}, bson.M{"$set": updateFields})
	if err != nil {
		return internalError(c, "failed to update llm provider")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "llm provider not found")
	}

	item, err := findLLMProviderByID(c)
	if err != nil {
		return internalError(c, "llm provider updated but could not be reloaded")
	}
	return c.JSON(dto.LLMProviderResponse{Success: true, Data: &item, Message: "llm provider updated successfully"})
}

func DeleteLLMProvider(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	database := providers.GetProviders(c).D
	providerModel := models.GetLLMProviderModel()
	result, err := database.DeleteOne(c.Context(), providerModel, bson.M{providerModel.IdKey: id})
	if err != nil {
		return internalError(c, "failed to delete llm provider")
	}
	if result.DeletedCount == 0 {
		return notFound(c, "llm provider not found")
	}

	return c.JSON(dto.LLMProviderResponse{Success: true, Message: "llm provider deleted successfully"})
}

func findLLMProviderByID(c *fiber.Ctx) (dto.LLMProviderItem, error) {
	provider, err := loadLLMProviderByID(c)
	if err != nil {
		return dto.LLMProviderItem{}, err
	}
	return toLLMProviderItem(provider), nil
}

func loadLLMProviderByID(c *fiber.Ctx) (models.LLMProvider, error) {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return models.LLMProvider{}, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	database := providers.GetProviders(c).D
	providerModel := models.GetLLMProviderModel()
	result := database.FindOne(c.Context(), providerModel, bson.M{providerModel.IdKey: id})

	var provider models.LLMProvider
	if err := result.Decode(&provider); err != nil {
		return models.LLMProvider{}, err
	}
	return provider, nil
}
