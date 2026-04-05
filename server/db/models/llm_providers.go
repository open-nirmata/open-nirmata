package models

import "time"

// LLMProvider represents a saved LLM provider configuration.
type LLMProvider struct {
	Id           string                 `bson:"id"`
	Name         string                 `bson:"name"`
	Provider     string                 `bson:"provider"`
	Description  string                 `bson:"description,omitempty"`
	Enabled      bool                   `bson:"enabled"`
	BaseURL      string                 `bson:"base_url,omitempty"`
	DefaultModel string                 `bson:"default_model,omitempty"`
	Organization string                 `bson:"organization,omitempty"`
	ProjectID    string                 `bson:"project_id,omitempty"`
	Auth         map[string]interface{} `bson:"auth,omitempty"`
	CreatedAt    *time.Time             `bson:"created_at,omitempty"`
	CreatedBy    string                 `bson:"created_by,omitempty"`
	UpdatedAt    *time.Time             `bson:"updated_at,omitempty"`
	UpdatedBy    string                 `bson:"updated_by,omitempty"`
}

// LLMProviderModel provides collection and field mappings for llm provider documents.
type LLMProviderModel struct {
	openNirmata
	IdKey           string
	NameKey         string
	ProviderKey     string
	DescriptionKey  string
	EnabledKey      string
	BaseURLKey      string
	DefaultModelKey string
	OrganizationKey string
	ProjectIDKey    string
	AuthKey         string
	CreatedAtKey    string
	CreatedByKey    string
	UpdatedAtKey    string
	UpdatedByKey    string
}

func (l LLMProviderModel) Name() string {
	return "llm_providers"
}

func GetLLMProviderModel() LLMProviderModel {
	return LLMProviderModel{
		IdKey:           "id",
		NameKey:         "name",
		ProviderKey:     "provider",
		DescriptionKey:  "description",
		EnabledKey:      "enabled",
		BaseURLKey:      "base_url",
		DefaultModelKey: "default_model",
		OrganizationKey: "organization",
		ProjectIDKey:    "project_id",
		AuthKey:         "auth",
		CreatedAtKey:    "created_at",
		CreatedByKey:    "created_by",
		UpdatedAtKey:    "updated_at",
		UpdatedByKey:    "updated_by",
	}
}
