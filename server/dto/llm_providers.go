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
