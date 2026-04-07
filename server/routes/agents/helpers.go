package agents

import (
	"errors"
	"fmt"
	"strings"

	"open-nirmata/db/models"
	"open-nirmata/dto"
	"open-nirmata/providers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var supportedAgentTypes = map[string]string{
	"chat": string(dto.AgentTypeChat),
}

func toAgentItem(agent models.Agent) dto.AgentItem {
	return dto.AgentItem{
		Id:           agent.Id,
		Name:         agent.Name,
		Description:  agent.Description,
		Enabled:      agent.Enabled,
		Type:         agent.Type,
		PromptFlowID: agent.PromptFlowID,
		CreatedAt:    agent.CreatedAt,
		UpdatedAt:    agent.UpdatedAt,
	}
}

func normalizeAgentType(agentType string) (string, bool) {
	normalizedKey := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(strings.ToLower(strings.TrimSpace(agentType)))
	normalizedType, ok := supportedAgentTypes[normalizedKey]
	return normalizedType, ok
}

func validateAgentRecord(c *fiber.Ctx, agent models.Agent) ([]string, error) {
	if strings.TrimSpace(agent.Name) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	normalizedType, ok := normalizeAgentType(agent.Type)
	if !ok {
		return nil, fiber.NewError(fiber.StatusBadRequest, "type must be: chat")
	}
	agent.Type = normalizedType

	if strings.TrimSpace(agent.PromptFlowID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "prompt_flow_id is required")
	}

	warnings := make([]string, 0)
	promptFlow, err := loadPromptFlowReference(c, agent.PromptFlowID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "prompt_flow_id references an unknown prompt flow")
		}
		return nil, err
	}
	if !promptFlow.Enabled {
		warnings = append(warnings, fmt.Sprintf("agent references disabled prompt flow %q", agent.PromptFlowID))
	}

	return warnings, nil
}

func loadPromptFlowReference(c *fiber.Ctx, id string) (models.PromptFlow, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.PromptFlow{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	flowModel := models.GetPromptFlowModel()
	result := serviceProvider.D.FindOne(c.Context(), flowModel, bson.M{flowModel.IdKey: strings.TrimSpace(id)})
	var flow models.PromptFlow
	if err := result.Decode(&flow); err != nil {
		return models.PromptFlow{}, err
	}
	return flow, nil
}

func findAgentByID(c *fiber.Ctx) (dto.AgentItem, error) {
	agent, err := loadAgentByID(c)
	if err != nil {
		return dto.AgentItem{}, err
	}
	return toAgentItem(agent), nil
}

func loadAgentByID(c *fiber.Ctx) (models.Agent, error) {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return models.Agent{}, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}
	return loadAgentByRecordID(c, id)
}

func loadAgentByRecordID(c *fiber.Ctx, id string) (models.Agent, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.Agent{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return models.Agent{}, fiber.NewError(fiber.StatusBadRequest, "agent id is required")
	}

	agentModel := models.GetAgentModel()
	result := serviceProvider.D.FindOne(c.Context(), agentModel, bson.M{agentModel.IdKey: trimmedID})

	var agent models.Agent
	if err := result.Decode(&agent); err != nil {
		return models.Agent{}, err
	}
	return agent, nil
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
