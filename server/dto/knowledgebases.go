package dto

import "time"

type KnowledgebaseProvider string

const (
	KnowledgebaseProviderMilvus      KnowledgebaseProvider = "milvus"
	KnowledgebaseProviderMixedbread  KnowledgebaseProvider = "mixedbread"
	KnowledgebaseProviderZeroEntropy KnowledgebaseProvider = "zeroentropy"
	KnowledgebaseProviderAlgolia     KnowledgebaseProvider = "algolia"
	KnowledgebaseProviderQdrant      KnowledgebaseProvider = "qdrant"
)

type CreateKnowledgebaseRequest struct {
	Name           string                 `json:"name"`
	Provider       string                 `json:"provider"`
	Description    string                 `json:"description,omitempty"`
	Enabled        *bool                  `json:"enabled,omitempty"`
	BaseURL        string                 `json:"base_url,omitempty"`
	IndexName      string                 `json:"index_name,omitempty"`
	Namespace      string                 `json:"namespace,omitempty"`
	EmbeddingModel string                 `json:"embedding_model,omitempty"`
	APIKey         string                 `json:"api_key,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	Auth           map[string]interface{} `json:"auth,omitempty"`
}

type UpdateKnowledgebaseRequest struct {
	Name           *string                 `json:"name,omitempty"`
	Provider       *string                 `json:"provider,omitempty"`
	Description    *string                 `json:"description,omitempty"`
	Enabled        *bool                   `json:"enabled,omitempty"`
	BaseURL        *string                 `json:"base_url,omitempty"`
	IndexName      *string                 `json:"index_name,omitempty"`
	Namespace      *string                 `json:"namespace,omitempty"`
	EmbeddingModel *string                 `json:"embedding_model,omitempty"`
	APIKey         *string                 `json:"api_key,omitempty"`
	Config         *map[string]interface{} `json:"config,omitempty"`
	Auth           *map[string]interface{} `json:"auth,omitempty"`
}

type KnowledgebaseItem struct {
	Id             string                 `json:"id"`
	Name           string                 `json:"name"`
	Provider       string                 `json:"provider"`
	Description    string                 `json:"description,omitempty"`
	Enabled        bool                   `json:"enabled"`
	BaseURL        string                 `json:"base_url,omitempty"`
	IndexName      string                 `json:"index_name,omitempty"`
	Namespace      string                 `json:"namespace,omitempty"`
	EmbeddingModel string                 `json:"embedding_model,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	AuthConfigured bool                   `json:"auth_configured"`
	CreatedAt      *time.Time             `json:"created_at,omitempty"`
	UpdatedAt      *time.Time             `json:"updated_at,omitempty"`
}

type KnowledgebaseResponse struct {
	Success bool               `json:"success"`
	Data    *KnowledgebaseItem `json:"data,omitempty"`
	Message string             `json:"message,omitempty"`
}

type KnowledgebaseListResponse struct {
	Success bool                `json:"success"`
	Data    []KnowledgebaseItem `json:"data"`
	Count   int                 `json:"count"`
}
