package promptflows

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

func ListPromptFlows(c *fiber.Ctx) error {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	flowModel := models.GetPromptFlowModel()
	filter := bson.M{}

	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		parsedEnabled, err := strconv.ParseBool(enabled)
		if err != nil {
			return badRequest(c, "invalid enabled value")
		}
		filter[flowModel.EnabledKey] = parsedEnabled
	}

	if searchText := strings.TrimSpace(c.Query("q")); searchText != "" {
		filter["$or"] = bson.A{
			bson.M{flowModel.NameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{flowModel.DescriptionKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{"stages.name": bson.M{"$regex": searchText, "$options": "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: flowModel.CreatedAtKey, Value: -1}})
	cursor, err := database.Find(c.Context(), flowModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list prompt flows")
	}
	defer cursor.Close(c.Context())

	items := make([]models.PromptFlow, 0)
	if err := cursor.All(c.Context(), &items); err != nil {
		return internalError(c, "failed to decode prompt flows")
	}

	response := make([]dto.PromptFlowItem, 0, len(items))
	for _, item := range items {
		response = append(response, toPromptFlowItem(item))
	}

	return c.JSON(dto.PromptFlowListResponse{Success: true, Data: response, Count: len(response)})
}

func GetPromptFlow(c *fiber.Ctx) error {
	item, err := findPromptFlowByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "prompt flow not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load prompt flow")
	}

	return c.JSON(dto.PromptFlowResponse{Success: true, Data: &item})
}

func ValidatePromptFlow(c *fiber.Ctx) error {
	var req dto.ValidatePromptFlowRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	flow := models.PromptFlow{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		Defaults:     toModelPromptFlowResources(req.Defaults),
		EntryStageID: strings.TrimSpace(req.EntryStageID),
		Stages:       toModelPromptFlowStages(req.Stages),
	}

	result, err := validatePromptFlowRecord(c, flow)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate prompt flow")
	}

	return c.JSON(dto.PromptFlowValidateResponse{Success: true, Data: result, Message: "prompt flow is valid"})
}

func CreatePromptFlow(c *fiber.Ctx) error {
	var req dto.CreatePromptFlowRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC()
	flow := models.PromptFlow{
		Id:           uuid.NewString(),
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		Defaults:     toModelPromptFlowResources(req.Defaults),
		EntryStageID: strings.TrimSpace(req.EntryStageID),
		Stages:       toModelPromptFlowStages(req.Stages),
		CreatedAt:    &now,
		CreatedBy:    "system",
		UpdatedAt:    &now,
		UpdatedBy:    "system",
	}

	validationResult, err := validatePromptFlowRecord(c, flow)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate prompt flow")
	}
	flow.EntryStageID = validationResult.EntryStageID

	database := serviceProvider.D
	flowModel := models.GetPromptFlowModel()
	count, err := database.CountDocuments(c.Context(), flowModel, bson.M{flowModel.NameKey: flow.Name})
	if err != nil {
		return internalError(c, "failed to validate prompt flow uniqueness")
	}
	if count > 0 {
		return badRequest(c, "prompt flow with the same name already exists")
	}

	if _, err := database.InsertOne(c.Context(), flowModel, flow); err != nil {
		return internalError(c, "failed to create prompt flow")
	}

	item := toPromptFlowItem(flow)
	return c.Status(fiber.StatusCreated).JSON(dto.PromptFlowResponse{Success: true, Data: &item, Message: "prompt flow created successfully"})
}

func UpdatePromptFlow(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	var req dto.UpdatePromptFlowRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	existing, err := loadPromptFlowByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "prompt flow not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load prompt flow")
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
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
		changed = true
	}
	if req.Enabled != nil {
		updated.Enabled = *req.Enabled
		changed = true
	}
	if req.Defaults != nil {
		updated.Defaults = toModelPromptFlowResources(req.Defaults)
		changed = true
	}
	if req.EntryStageID != nil {
		updated.EntryStageID = strings.TrimSpace(*req.EntryStageID)
		changed = true
	}
	if req.Stages != nil {
		updated.Stages = toModelPromptFlowStages(*req.Stages)
		changed = true
	}

	if !changed {
		return badRequest(c, "no fields to update")
	}

	validationResult, err := validatePromptFlowRecord(c, updated)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate prompt flow")
	}
	updated.EntryStageID = validationResult.EntryStageID

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	flowModel := models.GetPromptFlowModel()
	count, err := database.CountDocuments(c.Context(), flowModel, bson.M{
		flowModel.NameKey: updated.Name,
		flowModel.IdKey:   bson.M{"$ne": id},
	})
	if err != nil {
		return internalError(c, "failed to validate prompt flow uniqueness")
	}
	if count > 0 {
		return badRequest(c, "prompt flow with the same name already exists")
	}

	now := time.Now().UTC()
	updateFields := bson.M{
		flowModel.NameKey:         updated.Name,
		flowModel.DescriptionKey:  updated.Description,
		flowModel.EnabledKey:      updated.Enabled,
		flowModel.DefaultsKey:     updated.Defaults,
		flowModel.EntryStageIDKey: updated.EntryStageID,
		flowModel.StagesKey:       updated.Stages,
		flowModel.UpdatedAtKey:    &now,
		flowModel.UpdatedByKey:    "system",
	}

	result, err := database.UpdateOne(c.Context(), flowModel, bson.M{flowModel.IdKey: id}, bson.M{"$set": updateFields})
	if err != nil {
		return internalError(c, "failed to update prompt flow")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "prompt flow not found")
	}

	item, err := findPromptFlowByID(c)
	if err != nil {
		return internalError(c, "prompt flow updated but could not be reloaded")
	}
	return c.JSON(dto.PromptFlowResponse{Success: true, Data: &item, Message: "prompt flow updated successfully"})
}

func DeletePromptFlow(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	flowModel := models.GetPromptFlowModel()
	result, err := database.DeleteOne(c.Context(), flowModel, bson.M{flowModel.IdKey: id})
	if err != nil {
		return internalError(c, "failed to delete prompt flow")
	}
	if result.DeletedCount == 0 {
		return notFound(c, "prompt flow not found")
	}

	return c.JSON(dto.PromptFlowResponse{Success: true, Message: "prompt flow deleted successfully"})
}

func findPromptFlowByID(c *fiber.Ctx) (dto.PromptFlowItem, error) {
	flow, err := loadPromptFlowByID(c)
	if err != nil {
		return dto.PromptFlowItem{}, err
	}
	return toPromptFlowItem(flow), nil
}

func loadPromptFlowByID(c *fiber.Ctx) (models.PromptFlow, error) {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return models.PromptFlow{}, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}
	return loadPromptFlowByRecordID(c, id)
}

func loadPromptFlowByRecordID(c *fiber.Ctx, id string) (models.PromptFlow, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.PromptFlow{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return models.PromptFlow{}, fiber.NewError(fiber.StatusBadRequest, "prompt_flow_id is required")
	}

	flowModel := models.GetPromptFlowModel()
	result := serviceProvider.D.FindOne(c.Context(), flowModel, bson.M{flowModel.IdKey: trimmedID})

	var flow models.PromptFlow
	if err := result.Decode(&flow); err != nil {
		return models.PromptFlow{}, err
	}
	return flow, nil
}
