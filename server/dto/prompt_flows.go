package dto

import "time"

type PromptFlowStageType string

const (
	PromptFlowStageTypeChat      PromptFlowStageType = "chat"
	PromptFlowStageTypeTool      PromptFlowStageType = "tool"
	PromptFlowStageTypeRetrieval PromptFlowStageType = "retrieval"
	PromptFlowStageTypeRouter    PromptFlowStageType = "router"
)

type PromptFlowResources struct {
	LLMProviderID    string                 `json:"llm_provider_id,omitempty"`
	Model            string                 `json:"model,omitempty"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	ToolIDs          []string               `json:"tool_ids,omitempty"`
	KnowledgebaseIDs []string               `json:"knowledgebase_ids,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type PromptFlowTransition struct {
	Label         string `json:"label,omitempty"`
	Condition     string `json:"condition,omitempty"`
	TargetStageID string `json:"target_stage_id"`
}

type PromptFlowStage struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Overrides   *PromptFlowResources   `json:"overrides,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Transitions []PromptFlowTransition `json:"transitions,omitempty"`
}

type CreatePromptFlowRequest struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
	Defaults     *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID string               `json:"entry_stage_id,omitempty"`
	Stages       []PromptFlowStage    `json:"stages"`
}

type UpdatePromptFlowRequest struct {
	Name         *string              `json:"name,omitempty"`
	Description  *string              `json:"description,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
	Defaults     *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID *string              `json:"entry_stage_id,omitempty"`
	Stages       *[]PromptFlowStage   `json:"stages,omitempty"`
}

type ValidatePromptFlowRequest struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
	Defaults     *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID string               `json:"entry_stage_id,omitempty"`
	Stages       []PromptFlowStage    `json:"stages"`
}

type PromptFlowItem struct {
	Id           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Enabled      bool                 `json:"enabled"`
	Defaults     *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID string               `json:"entry_stage_id,omitempty"`
	Stages       []PromptFlowStage    `json:"stages,omitempty"`
	CreatedAt    *time.Time           `json:"created_at,omitempty"`
	UpdatedAt    *time.Time           `json:"updated_at,omitempty"`
}

type PromptFlowResponse struct {
	Success bool            `json:"success"`
	Data    *PromptFlowItem `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

type PromptFlowListResponse struct {
	Success bool             `json:"success"`
	Data    []PromptFlowItem `json:"data"`
	Count   int              `json:"count"`
}

type PromptFlowResolvedStage struct {
	Id              string               `json:"id"`
	Name            string               `json:"name"`
	Type            string               `json:"type"`
	Enabled         bool                 `json:"enabled"`
	Effective       *PromptFlowResources `json:"effective,omitempty"`
	TransitionCount int                  `json:"transition_count"`
}

type PromptFlowValidationResult struct {
	Valid        bool                      `json:"valid"`
	EntryStageID string                    `json:"entry_stage_id,omitempty"`
	Stages       []PromptFlowResolvedStage `json:"stages,omitempty"`
	Warnings     []string                  `json:"warnings,omitempty"`
}

type PromptFlowValidateResponse struct {
	Success bool                        `json:"success"`
	Data    *PromptFlowValidationResult `json:"data,omitempty"`
	Message string                      `json:"message,omitempty"`
}
