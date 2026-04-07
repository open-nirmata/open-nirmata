package agents

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

func ListAgents(c *fiber.Ctx) error {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	agentModel := models.GetAgentModel()
	filter := bson.M{}

	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		parsedEnabled, err := strconv.ParseBool(enabled)
		if err != nil {
			return badRequest(c, "invalid enabled value")
		}
		filter[agentModel.EnabledKey] = parsedEnabled
	}

	if searchText := strings.TrimSpace(c.Query("q")); searchText != "" {
		filter["$or"] = bson.A{
			bson.M{agentModel.NameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{agentModel.DescriptionKey: bson.M{"$regex": searchText, "$options": "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: agentModel.CreatedAtKey, Value: -1}})
	cursor, err := database.Find(c.Context(), agentModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list agents")
	}
	defer cursor.Close(c.Context())

	items := make([]models.Agent, 0)
	if err := cursor.All(c.Context(), &items); err != nil {
		return internalError(c, "failed to decode agents")
	}

	response := make([]dto.AgentItem, 0, len(items))
	for _, item := range items {
		response = append(response, toAgentItem(item))
	}

	return c.JSON(dto.AgentListResponse{Success: true, Data: response, Count: len(response)})
}

func GetAgent(c *fiber.Ctx) error {
	item, err := findAgentByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "agent not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load agent")
	}

	return c.JSON(dto.AgentResponse{Success: true, Data: &item})
}

func ValidateAgent(c *fiber.Ctx) error {
	var req dto.ValidateAgentRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	agent := models.Agent{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		Type:         strings.TrimSpace(req.Type),
		PromptFlowID: strings.TrimSpace(req.PromptFlowID),
	}

	warnings, err := validateAgentRecord(c, agent)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate agent")
	}

	return c.JSON(dto.AgentResponse{Success: true, Message: "agent is valid", Warnings: warnings})
}

func CreateAgent(c *fiber.Ctx) error {
	var req dto.CreateAgentRequest
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
	agent := models.Agent{
		Id:           uuid.NewString(),
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		Type:         strings.TrimSpace(req.Type),
		PromptFlowID: strings.TrimSpace(req.PromptFlowID),
		CreatedAt:    &now,
		CreatedBy:    "system",
		UpdatedAt:    &now,
		UpdatedBy:    "system",
	}

	warnings, err := validateAgentRecord(c, agent)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate agent")
	}

	normalizedType, _ := normalizeAgentType(agent.Type)
	agent.Type = normalizedType

	database := serviceProvider.D
	agentModel := models.GetAgentModel()
	count, err := database.CountDocuments(c.Context(), agentModel, bson.M{agentModel.NameKey: agent.Name})
	if err != nil {
		return internalError(c, "failed to validate agent uniqueness")
	}
	if count > 0 {
		return badRequest(c, "agent with the same name already exists")
	}

	if _, err := database.InsertOne(c.Context(), agentModel, agent); err != nil {
		return internalError(c, "failed to create agent")
	}

	item := toAgentItem(agent)
	return c.Status(fiber.StatusCreated).JSON(dto.AgentResponse{Success: true, Data: &item, Message: "agent created successfully", Warnings: warnings})
}

func UpdateAgent(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	var req dto.UpdateAgentRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	existing, err := loadAgentByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "agent not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load agent")
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
	if req.Type != nil {
		updated.Type = strings.TrimSpace(*req.Type)
		changed = true
	}
	if req.PromptFlowID != nil {
		updated.PromptFlowID = strings.TrimSpace(*req.PromptFlowID)
		if updated.PromptFlowID == "" {
			return badRequest(c, "prompt_flow_id cannot be empty")
		}
		changed = true
	}

	if !changed {
		return badRequest(c, "no fields to update")
	}

	warnings, err := validateAgentRecord(c, updated)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			if fiberErr.Code >= fiber.StatusInternalServerError {
				return internalError(c, fiberErr.Message)
			}
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate agent")
	}

	normalizedType, _ := normalizeAgentType(updated.Type)
	updated.Type = normalizedType

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	agentModel := models.GetAgentModel()
	count, err := database.CountDocuments(c.Context(), agentModel, bson.M{
		agentModel.NameKey: updated.Name,
		agentModel.IdKey:   bson.M{"$ne": id},
	})
	if err != nil {
		return internalError(c, "failed to validate agent uniqueness")
	}
	if count > 0 {
		return badRequest(c, "agent with the same name already exists")
	}

	now := time.Now().UTC()
	updateFields := bson.M{
		agentModel.NameKey:         updated.Name,
		agentModel.DescriptionKey:  updated.Description,
		agentModel.EnabledKey:      updated.Enabled,
		agentModel.TypeKey:         updated.Type,
		agentModel.PromptFlowIDKey: updated.PromptFlowID,
		agentModel.UpdatedAtKey:    &now,
		agentModel.UpdatedByKey:    "system",
	}

	result, err := database.UpdateOne(c.Context(), agentModel, bson.M{agentModel.IdKey: id}, bson.M{"$set": updateFields})
	if err != nil {
		return internalError(c, "failed to update agent")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "agent not found")
	}

	item, err := findAgentByID(c)
	if err != nil {
		return internalError(c, "agent updated but could not be reloaded")
	}
	return c.JSON(dto.AgentResponse{Success: true, Data: &item, Message: "agent updated successfully", Warnings: warnings})
}

func DeleteAgent(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return internalError(c, "database provider is not configured")
	}

	database := serviceProvider.D
	agentModel := models.GetAgentModel()
	result, err := database.DeleteOne(c.Context(), agentModel, bson.M{agentModel.IdKey: id})
	if err != nil {
		return internalError(c, "failed to delete agent")
	}
	if result.DeletedCount == 0 {
		return notFound(c, "agent not found")
	}

	return c.JSON(dto.AgentResponse{Success: true, Message: "agent deleted successfully"})
}
