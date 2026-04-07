package dto

import "time"

type AgentType string

const (
	AgentTypeChat AgentType = "chat"
)

type CreateAgentRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Type         string `json:"type"`
	PromptFlowID string `json:"prompt_flow_id"`
}

type UpdateAgentRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	Type         *string `json:"type,omitempty"`
	PromptFlowID *string `json:"prompt_flow_id,omitempty"`
}

type ValidateAgentRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Type         string `json:"type"`
	PromptFlowID string `json:"prompt_flow_id"`
}

type AgentItem struct {
	Id           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Enabled      bool       `json:"enabled"`
	Type         string     `json:"type"`
	PromptFlowID string     `json:"prompt_flow_id"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type AgentResponse struct {
	Success  bool       `json:"success"`
	Data     *AgentItem `json:"data,omitempty"`
	Message  string     `json:"message,omitempty"`
	Warnings []string   `json:"warnings,omitempty"`
}

type AgentListResponse struct {
	Success bool        `json:"success"`
	Data    []AgentItem `json:"data"`
	Count   int         `json:"count"`
}
