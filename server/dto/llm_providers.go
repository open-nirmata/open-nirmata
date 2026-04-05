package dto

import "time"

type CreateLLMProviderRequest struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	DefaultModel string `json:"default_model,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	Organization string `json:"organization,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
}

type UpdateLLMProviderRequest struct {
	Name         *string `json:"name,omitempty"`
	Provider     *string `json:"provider,omitempty"`
	Description  *string `json:"description,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	BaseURL      *string `json:"base_url,omitempty"`
	DefaultModel *string `json:"default_model,omitempty"`
	APIKey       *string `json:"api_key,omitempty"`
	Organization *string `json:"organization,omitempty"`
	ProjectID    *string `json:"project_id,omitempty"`
}

type LLMProviderItem struct {
	Id             string     `json:"id"`
	Name           string     `json:"name"`
	Provider       string     `json:"provider"`
	Description    string     `json:"description,omitempty"`
	Enabled        bool       `json:"enabled"`
	BaseURL        string     `json:"base_url,omitempty"`
	DefaultModel   string     `json:"default_model,omitempty"`
	Organization   string     `json:"organization,omitempty"`
	ProjectID      string     `json:"project_id,omitempty"`
	AuthConfigured bool       `json:"auth_configured"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type LLMProviderResponse struct {
	Success bool             `json:"success"`
	Data    *LLMProviderItem `json:"data,omitempty"`
	Message string           `json:"message,omitempty"`
}

type LLMProviderListResponse struct {
	Success bool              `json:"success"`
	Data    []LLMProviderItem `json:"data"`
	Count   int               `json:"count"`
}

type ListLLMProviderModelsRequest struct {
	LLMProviderID  string `json:"llm_provider_id,omitempty"`
	Provider       string `json:"provider,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
	Organization   string `json:"organization,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty"`
}

type LLMModelItem struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Provider         string                 `json:"provider"`
	Description      string                 `json:"description,omitempty"`
	OwnedBy          string                 `json:"owned_by,omitempty"`
	ContextWindow    int                    `json:"context_window,omitempty"`
	InputTokenLimit  int                    `json:"input_token_limit,omitempty"`
	OutputTokenLimit int                    `json:"output_token_limit,omitempty"`
	Capabilities     []string               `json:"capabilities,omitempty"`
	Raw              map[string]interface{} `json:"raw,omitempty"`
}

type ListLLMProviderModelsResponse struct {
	Success bool           `json:"success"`
	Data    []LLMModelItem `json:"data"`
	Count   int            `json:"count"`
	Message string         `json:"message,omitempty"`
}
