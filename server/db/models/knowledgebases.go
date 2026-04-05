package models

import "time"

// Knowledgebase represents a saved retrieval or vector-store provider configuration.
type Knowledgebase struct {
	Id             string                 `bson:"id"`
	Name           string                 `bson:"name"`
	Provider       string                 `bson:"provider"`
	Description    string                 `bson:"description,omitempty"`
	Enabled        bool                   `bson:"enabled"`
	BaseURL        string                 `bson:"base_url,omitempty"`
	IndexName      string                 `bson:"index_name,omitempty"`
	Namespace      string                 `bson:"namespace,omitempty"`
	EmbeddingModel string                 `bson:"embedding_model,omitempty"`
	Config         map[string]interface{} `bson:"config,omitempty"`
	Auth           map[string]interface{} `bson:"auth,omitempty"`
	CreatedAt      *time.Time             `bson:"created_at,omitempty"`
	CreatedBy      string                 `bson:"created_by,omitempty"`
	UpdatedAt      *time.Time             `bson:"updated_at,omitempty"`
	UpdatedBy      string                 `bson:"updated_by,omitempty"`
}

// KnowledgebaseModel provides collection and field mappings for knowledgebase documents.
type KnowledgebaseModel struct {
	openNirmata
	IdKey             string
	NameKey           string
	ProviderKey       string
	DescriptionKey    string
	EnabledKey        string
	BaseURLKey        string
	IndexNameKey      string
	NamespaceKey      string
	EmbeddingModelKey string
	ConfigKey         string
	AuthKey           string
	CreatedAtKey      string
	CreatedByKey      string
	UpdatedAtKey      string
	UpdatedByKey      string
}

func (k KnowledgebaseModel) Name() string {
	return "knowledgebases"
}

func GetKnowledgebaseModel() KnowledgebaseModel {
	return KnowledgebaseModel{
		IdKey:             "id",
		NameKey:           "name",
		ProviderKey:       "provider",
		DescriptionKey:    "description",
		EnabledKey:        "enabled",
		BaseURLKey:        "base_url",
		IndexNameKey:      "index_name",
		NamespaceKey:      "namespace",
		EmbeddingModelKey: "embedding_model",
		ConfigKey:         "config",
		AuthKey:           "auth",
		CreatedAtKey:      "created_at",
		CreatedByKey:      "created_by",
		UpdatedAtKey:      "updated_at",
		UpdatedByKey:      "updated_by",
	}
}
