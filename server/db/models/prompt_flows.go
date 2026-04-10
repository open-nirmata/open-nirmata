package models

import (
	"open-nirmata/dto"
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
	Source      string                 `bson:"source"`                // e.g., "system.usermessage", "stage1.output"
	Type        VariableMappingType    `bson:"type,omitempty"`        // direct, llm, template (default: direct)
	Default     interface{}            `bson:"default,omitempty"`     // default value if source not found
	Description string                 `bson:"description,omitempty"` // human-readable description
	Metadata    map[string]interface{} `bson:"metadata,omitempty"`    // additional metadata
}

// VariableDefinition defines an output variable from a stage
type VariableDefinition struct {
	Description string                 `bson:"description,omitempty"` // what this variable contains
	Type        string                 `bson:"type,omitempty"`        // string, object, array, etc.
	Source      string                 `bson:"source,omitempty"`      // how to extract (e.g., "response", "tool_result", "variables.key")
	Metadata    map[string]interface{} `bson:"metadata,omitempty"`    // additional metadata
}

type PromptFlowResources struct {
	LLMProviderID    string                 `bson:"llm_provider_id,omitempty"`
	Model            string                 `bson:"model,omitempty"`
	SystemPrompt     string                 `bson:"system_prompt,omitempty"`
	Temperature      *float64               `bson:"temperature,omitempty"`
	ToolIDs          []string               `bson:"tool_ids,omitempty"`
	KnowledgebaseIDs []string               `bson:"knowledgebase_ids,omitempty"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty"`
}

type PromptFlowTransition struct {
	Label         string `bson:"label,omitempty"`
	Condition     string `bson:"condition,omitempty"`
	TargetStageID string `bson:"target_stage_id"`
}

type PromptFlowStage struct {
	Id          string                         `bson:"id"`
	Name        string                         `bson:"name"`
	Type        dto.PromptFlowStageType        `bson:"type"`
	Description string                         `bson:"description,omitempty"`
	Prompt      string                         `bson:"prompt,omitempty"`
	Enabled     bool                           `bson:"enabled"`
	Overrides   *PromptFlowResources           `bson:"overrides,omitempty"`
	Config      map[string]interface{}         `bson:"config,omitempty"`
	Inputs      map[string]VariableMapping     `bson:"inputs,omitempty"`      // Input variable mappings
	Outputs     map[string]VariableDefinition  `bson:"outputs,omitempty"`     // Output variable definitions
	Transitions []PromptFlowTransition         `bson:"transitions,omitempty"`
	OnSuccess   string                         `bson:"on_success,omitempty"`
}

type PromptFlow struct {
	Id                         string               `bson:"id"`
	Name                       string               `bson:"name"`
	Description                string               `bson:"description,omitempty"`
	Enabled                    bool                 `bson:"enabled"`
	IncludeConversationHistory *bool                `bson:"include_conversation_history,omitempty"`
	Defaults                   *PromptFlowResources `bson:"defaults,omitempty"`
	EntryStageID               string               `bson:"entry_stage_id,omitempty"`
	Stages                     []PromptFlowStage    `bson:"stages,omitempty"`
	CreatedAt                  *time.Time           `bson:"created_at,omitempty"`
	CreatedBy                  string               `bson:"created_by,omitempty"`
	UpdatedAt                  *time.Time           `bson:"updated_at,omitempty"`
	UpdatedBy                  string               `bson:"updated_by,omitempty"`
}

type PromptFlowModel struct {
	openNirmata
	IdKey                         string
	NameKey                       string
	DescriptionKey                string
	EnabledKey                    string
	IncludeConversationHistoryKey string
	DefaultsKey                   string
	EntryStageIDKey               string
	StagesKey                     string
	CreatedAtKey                  string
	CreatedByKey                  string
	UpdatedAtKey                  string
	UpdatedByKey                  string
}

func (p PromptFlowModel) Name() string {
	return "prompt_flows"
}

func GetPromptFlowModel() PromptFlowModel {
	return PromptFlowModel{
		IdKey:                         "id",
		NameKey:                       "name",
		DescriptionKey:                "description",
		EnabledKey:                    "enabled",
		IncludeConversationHistoryKey: "include_conversation_history",
		DefaultsKey:                   "defaults",
		EntryStageIDKey:               "entry_stage_id",
		StagesKey:                     "stages",
		CreatedAtKey:                  "created_at",
		CreatedByKey:                  "created_by",
		UpdatedAtKey:                  "updated_at",
		UpdatedByKey:                  "updated_by",
	}
}
