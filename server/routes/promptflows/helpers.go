package promptflows

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

func toPromptFlowItem(flow models.PromptFlow) dto.PromptFlowItem {
	responseStages := make([]dto.PromptFlowStage, 0, len(flow.Stages))
	for _, stage := range flow.Stages {
		responseStages = append(responseStages, toPromptFlowStageItem(stage))
	}

	return dto.PromptFlowItem{
		Id:                         flow.Id,
		Name:                       flow.Name,
		Description:                flow.Description,
		Enabled:                    flow.Enabled,
		IncludeConversationHistory: cloneBoolPtr(flow.IncludeConversationHistory),
		Defaults:                   toPromptFlowResources(flow.Defaults),
		EntryStageID:               flow.EntryStageID,
		Stages:                     responseStages,
		CreatedAt:                  flow.CreatedAt,
		UpdatedAt:                  flow.UpdatedAt,
	}
}

func toPromptFlowStageItem(stage models.PromptFlowStage) dto.PromptFlowStage {
	enabled := stage.Enabled
	responseTransitions := make([]dto.PromptFlowTransition, 0, len(stage.Transitions))
	for _, transition := range stage.Transitions {
		responseTransitions = append(responseTransitions, dto.PromptFlowTransition{
			Label:         transition.Label,
			Condition:     transition.Condition,
			TargetStageID: transition.TargetStageID,
		})
	}

	return dto.PromptFlowStage{
		Id:          stage.Id,
		Name:        stage.Name,
		Type:        stage.Type,
		Description: stage.Description,
		Prompt:      stage.Prompt,
		Enabled:     &enabled,
		Overrides:   toPromptFlowResources(stage.Overrides),
		Config:      normalizeLooseMap(stage.Config),
		Transitions: responseTransitions,
		OnSuccess:   stage.OnSuccess,
	}
}

func toPromptFlowResources(resources *models.PromptFlowResources) *dto.PromptFlowResources {
	if resources == nil {
		return nil
	}

	response := &dto.PromptFlowResources{
		LLMProviderID:    strings.TrimSpace(resources.LLMProviderID),
		Model:            strings.TrimSpace(resources.Model),
		SystemPrompt:     strings.TrimSpace(resources.SystemPrompt),
		ToolIDs:          cloneStringList(resources.ToolIDs),
		KnowledgebaseIDs: cloneStringList(resources.KnowledgebaseIDs),
		Metadata:         normalizeLooseMap(resources.Metadata),
	}
	if resources.Temperature != nil {
		temperature := *resources.Temperature
		response.Temperature = &temperature
	}
	return response
}

func toModelPromptFlowResources(resources *dto.PromptFlowResources) *models.PromptFlowResources {
	if resources == nil {
		return nil
	}

	normalized := &models.PromptFlowResources{
		LLMProviderID:    strings.TrimSpace(resources.LLMProviderID),
		Model:            strings.TrimSpace(resources.Model),
		SystemPrompt:     strings.TrimSpace(resources.SystemPrompt),
		ToolIDs:          normalizeIDList(resources.ToolIDs),
		KnowledgebaseIDs: normalizeIDList(resources.KnowledgebaseIDs),
		Metadata:         normalizeLooseMap(resources.Metadata),
	}
	if resources.Temperature != nil {
		temperature := *resources.Temperature
		normalized.Temperature = &temperature
	}
	if isEmptyPromptFlowResources(normalized) {
		return nil
	}
	return normalized
}

func toModelPromptFlowStages(stages []dto.PromptFlowStage) []models.PromptFlowStage {
	if stages == nil {
		return nil
	}
	if len(stages) == 0 {
		return []models.PromptFlowStage{}
	}

	normalized := make([]models.PromptFlowStage, 0, len(stages))
	for _, stage := range stages {
		enabled := true
		if stage.Enabled != nil {
			enabled = *stage.Enabled
		}

		transitions := make([]models.PromptFlowTransition, 0, len(stage.Transitions))
		for _, transition := range stage.Transitions {
			transitions = append(transitions, models.PromptFlowTransition{
				Label:         strings.TrimSpace(transition.Label),
				Condition:     strings.TrimSpace(transition.Condition),
				TargetStageID: strings.TrimSpace(transition.TargetStageID),
			})
		}

		normalized = append(normalized, models.PromptFlowStage{
			Id:          strings.TrimSpace(stage.Id),
			Name:        strings.TrimSpace(stage.Name),
			Type:        stage.Type,
			Description: strings.TrimSpace(stage.Description),
			Prompt:      strings.TrimSpace(stage.Prompt),
			Enabled:     enabled,
			Overrides:   toModelPromptFlowResources(stage.Overrides),
			Config:      normalizeLooseMap(stage.Config),
			Transitions: transitions,
			OnSuccess:   strings.TrimSpace(stage.OnSuccess),
		})
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

func normalizeIDList(input []string) []string {
	if input == nil {
		return nil
	}
	if len(input) == 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

func cloneStringList(input []string) []string {
	if input == nil {
		return nil
	}
	cloned := make([]string, len(input))
	copy(cloned, input)
	return cloned
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func clonePromptFlowResources(resources *models.PromptFlowResources) *models.PromptFlowResources {
	if resources == nil {
		return nil
	}
	cloned := &models.PromptFlowResources{
		LLMProviderID:    strings.TrimSpace(resources.LLMProviderID),
		Model:            strings.TrimSpace(resources.Model),
		SystemPrompt:     strings.TrimSpace(resources.SystemPrompt),
		ToolIDs:          cloneStringList(resources.ToolIDs),
		KnowledgebaseIDs: cloneStringList(resources.KnowledgebaseIDs),
		Metadata:         normalizeLooseMap(resources.Metadata),
	}
	if resources.Temperature != nil {
		temperature := *resources.Temperature
		cloned.Temperature = &temperature
	}
	return cloned
}

func clonePromptFlowStages(stages []models.PromptFlowStage) []models.PromptFlowStage {
	if stages == nil {
		return nil
	}
	if len(stages) == 0 {
		return []models.PromptFlowStage{}
	}

	cloned := make([]models.PromptFlowStage, 0, len(stages))
	for _, stage := range stages {
		transitions := make([]models.PromptFlowTransition, len(stage.Transitions))
		copy(transitions, stage.Transitions)

		cloned = append(cloned, models.PromptFlowStage{
			Id:          strings.TrimSpace(stage.Id),
			Name:        strings.TrimSpace(stage.Name),
			Type:        stage.Type,
			Description: strings.TrimSpace(stage.Description),
			Prompt:      strings.TrimSpace(stage.Prompt),
			Enabled:     stage.Enabled,
			Overrides:   clonePromptFlowResources(stage.Overrides),
			Config:      normalizeLooseMap(stage.Config),
			Transitions: transitions,
			OnSuccess:   strings.TrimSpace(stage.OnSuccess),
		})
	}

	return cloned
}

func clonePromptFlow(flow models.PromptFlow) models.PromptFlow {
	return models.PromptFlow{
		Id:                         strings.TrimSpace(flow.Id),
		Name:                       strings.TrimSpace(flow.Name),
		Description:                strings.TrimSpace(flow.Description),
		Enabled:                    flow.Enabled,
		IncludeConversationHistory: cloneBoolPtr(flow.IncludeConversationHistory),
		Defaults:                   clonePromptFlowResources(flow.Defaults),
		EntryStageID:               strings.TrimSpace(flow.EntryStageID),
		Stages:                     clonePromptFlowStages(flow.Stages),
		CreatedAt:                  flow.CreatedAt,
		CreatedBy:                  strings.TrimSpace(flow.CreatedBy),
		UpdatedAt:                  flow.UpdatedAt,
		UpdatedBy:                  strings.TrimSpace(flow.UpdatedBy),
	}
}

func isEmptyPromptFlowResources(resources *models.PromptFlowResources) bool {
	if resources == nil {
		return true
	}
	if strings.TrimSpace(resources.LLMProviderID) != "" || strings.TrimSpace(resources.Model) != "" || strings.TrimSpace(resources.SystemPrompt) != "" || resources.Temperature != nil {
		return false
	}
	if resources.ToolIDs != nil || resources.KnowledgebaseIDs != nil {
		return false
	}
	return len(resources.Metadata) == 0
}

func mergePromptFlowResources(defaults *models.PromptFlowResources, overrides *models.PromptFlowResources) *models.PromptFlowResources {
	if defaults == nil && overrides == nil {
		return nil
	}

	merged := clonePromptFlowResources(defaults)
	if merged == nil {
		merged = &models.PromptFlowResources{}
	}

	if overrides != nil {
		if value := strings.TrimSpace(overrides.LLMProviderID); value != "" {
			merged.LLMProviderID = value
		}
		if value := strings.TrimSpace(overrides.Model); value != "" {
			merged.Model = value
		}
		if value := strings.TrimSpace(overrides.SystemPrompt); value != "" {
			merged.SystemPrompt = value
		}
		if overrides.Temperature != nil {
			temperature := *overrides.Temperature
			merged.Temperature = &temperature
		}
		if overrides.ToolIDs != nil {
			merged.ToolIDs = cloneStringList(overrides.ToolIDs)
		}
		if overrides.KnowledgebaseIDs != nil {
			merged.KnowledgebaseIDs = cloneStringList(overrides.KnowledgebaseIDs)
		}
		if overrides.Metadata != nil {
			merged.Metadata = normalizeLooseMap(overrides.Metadata)
		}
	}

	if isEmptyPromptFlowResources(merged) {
		return nil
	}
	return merged
}

func validatePromptFlowRecord(c *fiber.Ctx, flow models.PromptFlow) (*dto.PromptFlowValidationResult, error) {
	if strings.TrimSpace(flow.Name) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if len(flow.Stages) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "at least one stage is required")
	}

	stageIDs := map[string]struct{}{}
	for index := range flow.Stages {
		stage := &flow.Stages[index]
		if strings.TrimSpace(stage.Id) == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].id is required", index))
		}
		if strings.TrimSpace(stage.Name) == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].name is required", index))
		}
		if !stage.Type.IsValid() {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].type must be one of: llm, tool, retrieval, router, user_input, result", index))
		}
		if stage.Type.ShouldHaveOnSuccessTransition() && len(stage.OnSuccess) == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d] of type %q must specify on_success transition", index, stage.Type))
		}
		if _, exists := stageIDs[stage.Id]; exists {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("duplicate stage id %q", stage.Id))
		}
		stageIDs[stage.Id] = struct{}{}
	}

	entryStageID := strings.TrimSpace(flow.EntryStageID)
	if entryStageID == "" {
		entryStageID = flow.Stages[0].Id
	}
	if _, exists := stageIDs[entryStageID]; !exists {
		return nil, fiber.NewError(fiber.StatusBadRequest, "entry_stage_id must reference an existing stage")
	}

	for stageIndex, stage := range flow.Stages {
		if stage.Type == dto.PromptFlowStageTypeRouter && len(stage.Transitions) == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].transitions must contain at least one target for router stages", stageIndex))
		}
		for transitionIndex, transition := range stage.Transitions {
			if strings.TrimSpace(transition.TargetStageID) == "" {
				return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].transitions[%d].target_stage_id is required", stageIndex, transitionIndex))
			}
			if _, exists := stageIDs[strings.TrimSpace(transition.TargetStageID)]; !exists {
				return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("stages[%d].transitions[%d].target_stage_id references an unknown stage", stageIndex, transitionIndex))
			}
		}
	}

	warnings := make([]string, 0)
	defaultWarnings, err := validatePromptFlowResources(c, flow.Defaults, "defaults")
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, defaultWarnings...)

	resolvedStages := make([]dto.PromptFlowResolvedStage, 0, len(flow.Stages))
	for _, stage := range flow.Stages {
		stageWarnings, err := validatePromptFlowResources(c, stage.Overrides, fmt.Sprintf("stage %s", stage.Id))
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, stageWarnings...)

		resolvedStages = append(resolvedStages, dto.PromptFlowResolvedStage{
			Id:              stage.Id,
			Name:            stage.Name,
			Type:            stage.Type,
			Enabled:         stage.Enabled,
			Effective:       toPromptFlowResources(mergePromptFlowResources(flow.Defaults, stage.Overrides)),
			TransitionCount: len(stage.Transitions),
		})
	}

	return &dto.PromptFlowValidationResult{
		Valid:        true,
		EntryStageID: entryStageID,
		Stages:       resolvedStages,
		Warnings:     warnings,
	}, nil
}

func validatePromptFlowResources(c *fiber.Ctx, resources *models.PromptFlowResources, scope string) ([]string, error) {
	if resources == nil {
		return nil, nil
	}

	warnings := make([]string, 0)
	if strings.TrimSpace(resources.LLMProviderID) != "" {
		provider, err := loadLLMProviderReference(c, resources.LLMProviderID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s.llm_provider_id references an unknown llm provider", scope))
			}
			return nil, err
		}
		if !provider.Enabled {
			warnings = append(warnings, fmt.Sprintf("%s references disabled llm provider %q", scope, resources.LLMProviderID))
		}
	}

	for _, toolID := range resources.ToolIDs {
		tool, err := loadToolReference(c, toolID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s.tool_ids contains an unknown tool id %q", scope, toolID))
			}
			return nil, err
		}
		if !tool.Enabled {
			warnings = append(warnings, fmt.Sprintf("%s references disabled tool %q", scope, toolID))
		}
	}

	for _, knowledgebaseID := range resources.KnowledgebaseIDs {
		knowledgebase, err := loadKnowledgebaseReference(c, knowledgebaseID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s.knowledgebase_ids contains an unknown knowledgebase id %q", scope, knowledgebaseID))
			}
			return nil, err
		}
		if !knowledgebase.Enabled {
			warnings = append(warnings, fmt.Sprintf("%s references disabled knowledgebase %q", scope, knowledgebaseID))
		}
	}

	return warnings, nil
}

func loadLLMProviderReference(c *fiber.Ctx, id string) (models.LLMProvider, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.LLMProvider{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	providerModel := models.GetLLMProviderModel()
	result := serviceProvider.D.FindOne(c.Context(), providerModel, bson.M{providerModel.IdKey: strings.TrimSpace(id)})
	var provider models.LLMProvider
	if err := result.Decode(&provider); err != nil {
		return models.LLMProvider{}, err
	}
	return provider, nil
}

func loadToolReference(c *fiber.Ctx, id string) (models.Tool, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.Tool{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	toolModel := models.GetToolModel()
	result := serviceProvider.D.FindOne(c.Context(), toolModel, bson.M{toolModel.IdKey: strings.TrimSpace(id)})
	var tool models.Tool
	if err := result.Decode(&tool); err != nil {
		return models.Tool{}, err
	}
	return tool, nil
}

func loadKnowledgebaseReference(c *fiber.Ctx, id string) (models.Knowledgebase, error) {
	serviceProvider := providers.GetProviders(c)
	if serviceProvider == nil || serviceProvider.D == nil {
		return models.Knowledgebase{}, fiber.NewError(fiber.StatusInternalServerError, "database provider is not configured")
	}

	knowledgebaseModel := models.GetKnowledgebaseModel()
	result := serviceProvider.D.FindOne(c.Context(), knowledgebaseModel, bson.M{knowledgebaseModel.IdKey: strings.TrimSpace(id)})
	var knowledgebase models.Knowledgebase
	if err := result.Decode(&knowledgebase); err != nil {
		return models.Knowledgebase{}, err
	}
	return knowledgebase, nil
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
