package tools

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

func ListTools(c *fiber.Ctx) error {
	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	filter := bson.M{}

	if toolType := strings.TrimSpace(c.Query("type")); toolType != "" {
		normalizedType, ok := normalizeToolType(toolType)
		if !ok {
			return badRequest(c, "invalid tool type")
		}
		filter[toolModel.TypeKey] = normalizedType
	}

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		normalizedProvider, err := normalizeProvider(string(dto.ToolTypeLLM), provider)
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				return badRequest(c, fiberErr.Message)
			}
			return badRequest(c, "invalid provider")
		}
		filter[toolModel.ProviderKey] = normalizedProvider
	}

	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		parsedEnabled, err := strconv.ParseBool(enabled)
		if err != nil {
			return badRequest(c, "invalid enabled value")
		}
		filter[toolModel.EnabledKey] = parsedEnabled
	}

	if searchText := strings.TrimSpace(c.Query("q")); searchText != "" {
		filter["$or"] = bson.A{
			bson.M{toolModel.NameKey: bson.M{"$regex": searchText, "$options": "i"}},
			bson.M{toolModel.DescriptionKey: bson.M{"$regex": searchText, "$options": "i"}},
		}
	}

	findOptions := options.Find().SetSort(bson.D{{Key: toolModel.CreatedAtKey, Value: -1}})
	cursor, err := database.Find(c.Context(), toolModel, filter, findOptions)
	if err != nil {
		return internalError(c, "failed to list tools")
	}
	defer cursor.Close(c.Context())

	items := make([]models.Tool, 0)
	if err := cursor.All(c.Context(), &items); err != nil {
		return internalError(c, "failed to decode tools")
	}

	response := make([]dto.ToolItem, 0, len(items))
	for _, item := range items {
		response = append(response, toToolItem(item))
	}

	return c.JSON(dto.ToolListResponse{Success: true, Data: response, Count: len(response)})
}

func GetTool(c *fiber.Ctx) error {
	item, err := findToolByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "tool not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load tool")
	}

	return c.JSON(dto.ToolResponse{Success: true, Data: &item})
}

func TestMCPTool(c *fiber.Ctx) error {
	var req dto.TestMCPToolRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}
	if req.Config == nil {
		return badRequest(c, "config is required")
	}

	req.Config = normalizeToolConfig(req.Config)
	if err := validateMCPConfig(req.Config); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate mcp config")
	}

	timeout, err := resolveMCPTestTimeout(req.TimeoutSeconds)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to resolve timeout")
	}

	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.S == nil || serviceProvider.S.MCP == nil {
		return internalError(c, "mcp service is not configured")
	}

	result, err := serviceProvider.S.MCP.ListTools(c.UserContext(), req.Config, timeout)
	if err != nil {
		return badGateway(c, "failed to list tools from mcp server: "+err.Error())
	}

	return c.JSON(dto.TestMCPToolResponse{
		Success: true,
		Data:    result,
		Message: "mcp tools fetched successfully",
	})
}

func CreateTool(c *fiber.Ctx) error {
	var req dto.CreateToolRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return badRequest(c, "name is required")
	}

	toolType, ok := normalizeToolType(req.Type)
	if !ok {
		return badRequest(c, "invalid tool type")
	}

	provider, err := normalizeProvider(toolType, req.Provider)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return badRequest(c, "invalid provider")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC()
	tool := models.Tool{
		Id:          uuid.NewString(),
		Name:        name,
		Type:        toolType,
		Provider:    provider,
		Description: strings.TrimSpace(req.Description),
		Enabled:     enabled,
		Tags:        normalizeTags(req.Tags),
		Config:      normalizeToolConfig(req.Config),
		Auth:        normalizeLooseMap(req.Auth),
		CreatedAt:   &now,
		CreatedBy:   "system",
		UpdatedAt:   &now,
		UpdatedBy:   "system",
	}

	if err := validateToolRecord(tool); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate tool")
	}

	if tool.Type == string(dto.ToolTypeMCP) {
		if err := refreshMCPToolMetadata(c, &tool, nil); err != nil {
			return handleMCPRefreshError(c, err)
		}
	}

	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	if _, err := database.InsertOne(c.Context(), toolModel, tool); err != nil {
		return internalError(c, "failed to create tool")
	}

	item := toToolItem(tool)
	return c.Status(fiber.StatusCreated).JSON(dto.ToolResponse{Success: true, Data: &item, Message: "tool created successfully"})
}

func UpdateTool(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	var req dto.UpdateToolRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	existing, err := loadToolByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "tool not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load tool")
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
	if req.Type != nil {
		toolType, ok := normalizeToolType(*req.Type)
		if !ok {
			return badRequest(c, "invalid tool type")
		}
		updated.Type = toolType
		changed = true
	}
	if req.Provider != nil {
		updated.Provider = strings.TrimSpace(*req.Provider)
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
	if req.Tags != nil {
		updated.Tags = normalizeTags(*req.Tags)
		changed = true
	}
	if req.Config != nil {
		updated.Config = normalizeToolConfig(req.Config)
		changed = true
	}
	if req.Auth != nil {
		updated.Auth = normalizeLooseMap(*req.Auth)
		changed = true
	}

	if !changed {
		return badRequest(c, "no fields to update")
	}

	updated.Provider, err = normalizeProvider(updated.Type, updated.Provider)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return badRequest(c, "invalid provider")
	}

	if err := validateToolRecord(updated); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to validate tool")
	}

	if updated.Type == string(dto.ToolTypeMCP) {
		if err := refreshMCPToolMetadata(c, &updated, nil); err != nil {
			return handleMCPRefreshError(c, err)
		}
	}

	now := time.Now().UTC()
	updated.UpdatedAt = &now
	updated.UpdatedBy = "system"

	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	updateFields := buildToolUpdateFields(updated)

	result, err := database.UpdateOne(c.Context(), toolModel, bson.M{toolModel.IdKey: id}, bson.M{"$set": updateFields})
	if err != nil {
		return internalError(c, "failed to update tool")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "tool not found")
	}

	item := toToolItem(updated)
	return c.JSON(dto.ToolResponse{Success: true, Data: &item, Message: "tool updated successfully"})
}

func RefreshTool(c *fiber.Ctx) error {
	var req dto.RefreshToolRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}
	}

	existing, err := loadToolByID(c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return notFound(c, "tool not found")
		}
		if fiberErr, ok := err.(*fiber.Error); ok {
			return badRequest(c, fiberErr.Message)
		}
		return internalError(c, "failed to load tool")
	}

	if existing.Type != string(dto.ToolTypeMCP) {
		return badRequest(c, "refresh is only supported for mcp tools")
	}

	if err := refreshMCPToolMetadata(c, &existing, req.TimeoutSeconds); err != nil {
		return handleMCPRefreshError(c, err)
	}

	now := time.Now().UTC()
	existing.UpdatedAt = &now
	existing.UpdatedBy = "system"

	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	result, err := database.UpdateOne(c.Context(), toolModel, bson.M{toolModel.IdKey: existing.Id}, bson.M{"$set": buildToolUpdateFields(existing)})
	if err != nil {
		return internalError(c, "failed to refresh tool")
	}
	if result.MatchedCount == 0 {
		return notFound(c, "tool not found")
	}

	item := toToolItem(existing)
	return c.JSON(dto.ToolResponse{Success: true, Data: &item, Message: "mcp tool refreshed successfully"})
}

func DeleteTool(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return badRequest(c, "id is required")
	}

	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	result, err := database.DeleteOne(c.Context(), toolModel, bson.M{toolModel.IdKey: id})
	if err != nil {
		return internalError(c, "failed to delete tool")
	}
	if result.DeletedCount == 0 {
		return notFound(c, "tool not found")
	}

	return c.JSON(dto.ToolResponse{Success: true, Message: "tool deleted successfully"})
}

func loadToolByID(c *fiber.Ctx) (models.Tool, error) {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return models.Tool{}, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	database := providers.GetProviders(c).D
	toolModel := models.GetToolModel()
	result := database.FindOne(c.Context(), toolModel, bson.M{toolModel.IdKey: id})

	var tool models.Tool
	if err := result.Decode(&tool); err != nil {
		return models.Tool{}, err
	}
	return tool, nil
}

func findToolByID(c *fiber.Ctx) (dto.ToolItem, error) {
	tool, err := loadToolByID(c)
	if err != nil {
		return dto.ToolItem{}, err
	}
	return toToolItem(tool), nil
}

func buildToolUpdateFields(tool models.Tool) bson.M {
	toolModel := models.GetToolModel()
	return bson.M{
		toolModel.NameKey:        tool.Name,
		toolModel.TypeKey:        tool.Type,
		toolModel.ProviderKey:    tool.Provider,
		toolModel.DescriptionKey: tool.Description,
		toolModel.EnabledKey:     tool.Enabled,
		toolModel.ToolsKey:       tool.Tools,
		toolModel.TagsKey:        tool.Tags,
		toolModel.ConfigKey:      tool.Config,
		toolModel.AuthKey:        tool.Auth,
		toolModel.UpdatedAtKey:   tool.UpdatedAt,
		toolModel.UpdatedByKey:   tool.UpdatedBy,
	}
}

func handleMCPRefreshError(c *fiber.Ctx, err error) error {
	if fiberErr, ok := err.(*fiber.Error); ok {
		switch fiberErr.Code {
		case fiber.StatusNotFound:
			return notFound(c, fiberErr.Message)
		case fiber.StatusBadRequest:
			return badRequest(c, fiberErr.Message)
		default:
			return c.Status(fiberErr.Code).JSON(dto.ErrorResponse{Success: false, Message: fiberErr.Message})
		}
	}
	return badGateway(c, "failed to fetch tools from mcp server: "+err.Error())
}
