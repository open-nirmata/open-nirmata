package models

import "time"

// Agent represents a chat agent that references a prompt flow.
type Agent struct {
	Id           string     `bson:"id"`
	Name         string     `bson:"name"`
	Description  string     `bson:"description,omitempty"`
	Enabled      bool       `bson:"enabled"`
	Type         string     `bson:"type"`
	PromptFlowID string     `bson:"prompt_flow_id"`
	CreatedAt    *time.Time `bson:"created_at,omitempty"`
	CreatedBy    string     `bson:"created_by,omitempty"`
	UpdatedAt    *time.Time `bson:"updated_at,omitempty"`
	UpdatedBy    string     `bson:"updated_by,omitempty"`
}

// AgentModel provides collection and field mappings for agent documents.
type AgentModel struct {
	openNirmata
	IdKey           string
	NameKey         string
	DescriptionKey  string
	EnabledKey      string
	TypeKey         string
	PromptFlowIDKey string
	CreatedAtKey    string
	CreatedByKey    string
	UpdatedAtKey    string
	UpdatedByKey    string
}

func (a AgentModel) Name() string {
	return "agents"
}

func GetAgentModel() AgentModel {
	return AgentModel{
		IdKey:           "id",
		NameKey:         "name",
		DescriptionKey:  "description",
		EnabledKey:      "enabled",
		TypeKey:         "type",
		PromptFlowIDKey: "prompt_flow_id",
		CreatedAtKey:    "created_at",
		CreatedByKey:    "created_by",
		UpdatedAtKey:    "updated_at",
		UpdatedByKey:    "updated_by",
	}
}
