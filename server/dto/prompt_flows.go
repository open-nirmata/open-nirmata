package dto

import (
	"time"
)

// VariableMappingType defines how a variable should be mapped
type VariableMappingType string

const (
	VariableMappingTypeDirect   VariableMappingType = "direct"   // Copy value directly
	VariableMappingTypeLLM      VariableMappingType = "llm"      // Prompt handles variable usage
	VariableMappingTypeTemplate VariableMappingType = "template" // Use Go template rendering
)

// VariableMapping defines how an input variable should be populated
type VariableMapping struct {
	Source      string                 `json:"source"`                // e.g., "system.usermessage", "stage1.output"
	Type        VariableMappingType    `json:"type,omitempty"`        // direct, llm, template (default: direct)
	Default     interface{}            `json:"default,omitempty"`     // default value if source not found
	Description string                 `json:"description,omitempty"` // human-readable description
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // additional metadata
}

// VariableDefinition defines an output variable from a stage
type VariableDefinition struct {
	Description string                 `json:"description,omitempty"` // what this variable contains
	Type        string                 `json:"type,omitempty"`        // string, object, array, etc.
	Source      string                 `json:"source,omitempty"`      // how to extract (e.g., "response", "tool_result", "variables.key")
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // additional metadata
}

type PromptFlowStageType string

const (
	PromptFlowStageTypeLLM       PromptFlowStageType = "llm"
	PromptFlowStageTypeTool      PromptFlowStageType = "tool"
	PromptFlowStageTypeRetrieval PromptFlowStageType = "retrieval"
	PromptFlowStageTypeRouter    PromptFlowStageType = "router"
	PromptFlowStageTypeResult    PromptFlowStageType = "result"
)

func (p PromptFlowStageType) IsValid() bool {
	switch p {
	case PromptFlowStageTypeLLM, PromptFlowStageTypeTool, PromptFlowStageTypeRetrieval, PromptFlowStageTypeRouter, PromptFlowStageTypeResult:
		return true
	default:
		return false
	}
}

func (p PromptFlowStageType) ShouldHaveOnSuccessTransition() bool {
	switch p {
	case PromptFlowStageTypeLLM, PromptFlowStageTypeRetrieval, PromptFlowStageTypeTool:
		return true
	default:
		return false
	}
}

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
	Id          string                        `json:"id"`
	Name        string                        `json:"name"`
	Type        PromptFlowStageType           `json:"type"`
	Description string                        `json:"description,omitempty"`
	Prompt      string                        `json:"prompt,omitempty"`
	Enabled     *bool                         `json:"enabled,omitempty"`
	Overrides   *PromptFlowResources          `json:"overrides,omitempty"`
	Config      map[string]interface{}        `json:"config,omitempty"`
	Inputs      map[string]VariableMapping    `json:"inputs,omitempty"`      // Input variable mappings
	Outputs     map[string]VariableDefinition `json:"outputs,omitempty"`     // Output variable definitions
	Transitions []PromptFlowTransition        `json:"transitions,omitempty"`
	OnSuccess   string                        `json:"on_success,omitempty"`
}

type CreatePromptFlowRequest struct {
	Name                       string               `json:"name"`
	Description                string               `json:"description,omitempty"`
	Enabled                    *bool                `json:"enabled,omitempty"`
	IncludeConversationHistory *bool                `json:"include_conversation_history,omitempty"`
	Defaults                   *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID               string               `json:"entry_stage_id,omitempty"`
	Stages                     []PromptFlowStage    `json:"stages"`
}

type CopyPromptFlowRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type UpdatePromptFlowRequest struct {
	Name                       *string              `json:"name,omitempty"`
	Description                *string              `json:"description,omitempty"`
	Enabled                    *bool                `json:"enabled,omitempty"`
	IncludeConversationHistory *bool                `json:"include_conversation_history,omitempty"`
	Defaults                   *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID               *string              `json:"entry_stage_id,omitempty"`
	Stages                     *[]PromptFlowStage   `json:"stages,omitempty"`
}

type ValidatePromptFlowRequest struct {
	Name                       string               `json:"name"`
	Description                string               `json:"description,omitempty"`
	Enabled                    *bool                `json:"enabled,omitempty"`
	IncludeConversationHistory *bool                `json:"include_conversation_history,omitempty"`
	Defaults                   *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID               string               `json:"entry_stage_id,omitempty"`
	Stages                     []PromptFlowStage    `json:"stages"`
}

type PromptFlowItem struct {
	Id                         string               `json:"id"`
	Name                       string               `json:"name"`
	Description                string               `json:"description,omitempty"`
	Enabled                    bool                 `json:"enabled"`
	IncludeConversationHistory *bool                `json:"include_conversation_history,omitempty"`
	Defaults                   *PromptFlowResources `json:"defaults,omitempty"`
	EntryStageID               string               `json:"entry_stage_id,omitempty"`
	Stages                     []PromptFlowStage    `json:"stages,omitempty"`
	CreatedAt                  *time.Time           `json:"created_at,omitempty"`
	UpdatedAt                  *time.Time           `json:"updated_at,omitempty"`
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
	Type            PromptFlowStageType  `json:"type"`
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
