package knowledgebases

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

func ListKnowledgebases(c *fiber.Ctx) error {
	database := providers.GetProviders(c).D
	knowledgebaseModel := models.GetKnowledgebaseModel()
	filter := bson.M{}

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		normalizedProvider, ok := normalizeKnowledgebaseProvider(provider)
		if !ok {
			return badRequest(c, "invalid provider; supported values: milvus, mixedbread, zeroentropy, algolia, qdrant")
		}
		filter[knowledgebaseModel.ProviderKey] = normalizedProvider
	}

	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		parsedEnabled, err := strconv.ParseBool(enabled)
		if err != nil {
			return badRequest(c, "invalid enabled value")
		}
		filter[knowledgebaseModel.EnabledKey] = parsedEnabled
	}

	if searchText := strings.TrimSpace(c.Query("q")); searchText != "" {
		filter["$or"] = bson.A{
			bson.M{knowledgebaseModel.NameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{knowledgebaseModel.DescriptionKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{knowledgebaseModel.IndexNameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{knowledgebaseModel.NamespaceKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{knowledgebaseModel.EmbeddingModelKey: bson.M{"$regex": searchText, "$options": "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: knowledgebaseModel.CreatedAtKey, Value: -1}})
	cursor, err := database.Find(c.Context(), knowledgebaseModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list knowledgebases")
	}
	defer cursor.Close(c.Context())

	items := make([]models.Knowledgebase, 0)
	if err := cursor.All(c.Context(), &items); err != nil {
		return internalError(c, "failed to decode knowledgebases")
	}

	response := make([]dto.KnowledgebaseItem, 0, len(items))
	for _, item := range items {
		response = append(response, toKnowledgebaseItem(item))
	}

	return c.JSON(dto.KnowledgebaseListResponse{Success: true, Data: response, Count: len(response)})
}

func GetKnowledgebase(c *fiber.Ctx) error {
	item, err := findKnowledgebaseByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "knowledgebase not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load knowledgebase")
	}

	return c.JSON(dto.KnowledgebaseResponse{Success: true, Data: &item})
}

func CreateKnowledgebase(c *fiber.Ctx) error {
	var req dto.CreateKnowledgebaseRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return badRequest(c, "name is required")
	}

	normalizedProvider, ok := normalizeKnowledgebaseProvider(req.Provider)
	if !ok {
		return badRequest(c, "invalid provider; supported values: milvus, mixedbread, zeroentropy, algolia, qdrant")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC()
	knowledgebase := models.Knowledgebase{
		Id:             uuid.NewString(),
		Name:           name,
		Provider:       normalizedProvider,
		Description:    strings.TrimSpace(req.Description),
		Enabled:        enabled,
		BaseURL:        strings.TrimSpace(req.BaseURL),
		IndexName:      strings.TrimSpace(req.IndexName),
		Namespace:      strings.TrimSpace(req.Namespace),
		EmbeddingModel: strings.TrimSpace(req.EmbeddingModel),
		Config:         normalizeLooseMap(req.Config),
		Auth:           mergeKnowledgebaseAuth(nil, &req.APIKey, &req.Auth),
		CreatedAt:      &now,
		CreatedBy:      "system",
		UpdatedAt:      &now,
		UpdatedBy:      "system",
	}

	if err := validateKnowledgebaseRecord(knowledgebase); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate knowledgebase")
	}

	database := providers.GetProviders(c).D
	knowledgebaseModel := models.GetKnowledgebaseModel()
	count, err := database.CountDocuments(c.Context(), knowledgebaseModel, bson.M{
		knowledgebaseModel.NameKey:     knowledgebase.Name,
		knowledgebaseModel.ProviderKey: knowledgebase.Provider,
	})
	if err != nil {
		return internalError(c, "failed to validate knowledgebase uniqueness")
	}
	if count > 0 {
		return badRequest(c, "knowledgebase with the same name and provider already exists")
	}

	if _, err := database.InsertOne(c.Context(), knowledgebaseModel, knowledgebase); err != nil {
		return internalError(c, "failed to create knowledgebase")
	}

	item := toKnowledgebaseItem(knowledgebase)
	return c.Status(fiber.StatusCreated).JSON(dto.KnowledgebaseResponse{Success: true, Data: &item, Message: "knowledgebase created successfully"})
}

func UpdateKnowledgebase(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	var req dto.UpdateKnowledgebaseRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	existing, err := loadKnowledgebaseByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "knowledgebase not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load knowledgebase")
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
		normalizedProvider, ok := normalizeKnowledgebaseProvider(*req.Provider)
		if !ok {
			return badRequest(c, "invalid provider; supported values: milvus, mixedbread, zeroentropy, algolia, qdrant")
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
	if req.IndexName != nil {
		updated.IndexName = strings.TrimSpace(*req.IndexName)
		changed = true
	}
	if req.Namespace != nil {
		updated.Namespace = strings.TrimSpace(*req.Namespace)
		changed = true
	}
	if req.EmbeddingModel != nil {
		updated.EmbeddingModel = strings.TrimSpace(*req.EmbeddingModel)
		changed = true
	}
	if req.Config != nil {
		updated.Config = normalizeLooseMap(*req.Config)
		changed = true
	}
	if req.Auth != nil || req.APIKey != nil {
		updated.Auth = mergeKnowledgebaseAuth(updated.Auth, req.APIKey, req.Auth)
		changed = true
	}

	if !changed {
		return badRequest(c, "no fields to update")
	}

	if err := validateKnowledgebaseRecord(updated); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate knowledgebase")
	}

	database := providers.GetProviders(c).D
	knowledgebaseModel := models.GetKnowledgebaseModel()
	count, err := database.CountDocuments(c.Context(), knowledgebaseModel, bson.M{
		knowledgebaseModel.NameKey:     updated.Name,
		knowledgebaseModel.ProviderKey: updated.Provider,
		knowledgebaseModel.IdKey:       bson.M{"$ne": id},
	})
	if err != nil {
		return internalError(c, "failed to validate knowledgebase uniqueness")
	}
	if count > 0 {
		return badRequest(c, "knowledgebase with the same name and provider already exists")
	}

	now := time.Now().UTC()
	updateFields := bson.M{
		knowledgebaseModel.NameKey:           updated.Name,
		knowledgebaseModel.ProviderKey:       updated.Provider,
		knowledgebaseModel.DescriptionKey:    updated.Description,
		knowledgebaseModel.EnabledKey:        updated.Enabled,
		knowledgebaseModel.BaseURLKey:        updated.BaseURL,
		knowledgebaseModel.IndexNameKey:      updated.IndexName,
		knowledgebaseModel.NamespaceKey:      updated.Namespace,
		knowledgebaseModel.EmbeddingModelKey: updated.EmbeddingModel,
		knowledgebaseModel.ConfigKey:         updated.Config,
		knowledgebaseModel.AuthKey:           updated.Auth,
		knowledgebaseModel.UpdatedAtKey:      &now,
		knowledgebaseModel.UpdatedByKey:      "system",
	}

	result, err := database.UpdateOne(c.Context(), knowledgebaseModel, bson.M{knowledgebaseModel.IdKey: id}, bson.M{"$set": updateFields})
	if err != nil {
		return internalError(c, "failed to update knowledgebase")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "knowledgebase not found")
	}

	item, err := findKnowledgebaseByID(c)
	if err != nil {
		return internalError(c, "knowledgebase updated but could not be reloaded")
	}
	return c.JSON(dto.KnowledgebaseResponse{Success: true, Data: &item, Message: "knowledgebase updated successfully"})
}

func DeleteKnowledgebase(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	database := providers.GetProviders(c).D
	knowledgebaseModel := models.GetKnowledgebaseModel()
	result, err := database.DeleteOne(c.Context(), knowledgebaseModel, bson.M{knowledgebaseModel.IdKey: id})
	if err != nil {
		return internalError(c, "failed to delete knowledgebase")
	}
	if result.DeletedCount == 0 {
		return notFound(c, "knowledgebase not found")
	}

	return c.JSON(dto.KnowledgebaseResponse{Success: true, Message: "knowledgebase deleted successfully"})
}

func findKnowledgebaseByID(c *fiber.Ctx) (dto.KnowledgebaseItem, error) {
	knowledgebase, err := loadKnowledgebaseByID(c)
	if err != nil {
		return dto.KnowledgebaseItem{}, err
	}
	return toKnowledgebaseItem(knowledgebase), nil
}

func loadKnowledgebaseByID(c *fiber.Ctx) (models.Knowledgebase, error) {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return models.Knowledgebase{}, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	database := providers.GetProviders(c).D
	knowledgebaseModel := models.GetKnowledgebaseModel()
	result := database.FindOne(c.Context(), knowledgebaseModel, bson.M{knowledgebaseModel.IdKey: id})

	var knowledgebase models.Knowledgebase
	if err := result.Decode(&knowledgebase); err != nil {
		return models.Knowledgebase{}, err
	}
	return knowledgebase, nil
}
