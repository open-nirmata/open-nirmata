package models

import (
	"time"

	"open-nirmata/dto"
)

// Tool represents a tool definition stored for the agent builder.
type Tool struct {
	Id          string                  `bson:"id"`
	Name        string                  `bson:"name"`
	Type        string                  `bson:"type"`
	Provider    string                  `bson:"provider,omitempty"`
	Description string                  `bson:"description,omitempty"`
	Tools       []dto.MCPDiscoveredTool `bson:"tools,omitempty"` // for composite tools, list of underlying tool ids
	Enabled     bool                    `bson:"enabled"`
	Tags        []string                `bson:"tags,omitempty"`
	Config      *dto.ToolConfig         `bson:"config,omitempty"`
	Auth        map[string]interface{}  `bson:"auth,omitempty"`
	CreatedAt   *time.Time              `bson:"created_at,omitempty"`
	CreatedBy   string                  `bson:"created_by,omitempty"`
	UpdatedAt   *time.Time              `bson:"updated_at,omitempty"`
	UpdatedBy   string                  `bson:"updated_by,omitempty"`
}

// ToolModel provides collection and field mappings for tool documents.
type ToolModel struct {
	openNirmata
	IdKey          string
	NameKey        string
	TypeKey        string
	ProviderKey    string
	DescriptionKey string
	EnabledKey     string
	ToolsKey       string
	TagsKey        string
	ConfigKey      string
	AuthKey        string
	CreatedAtKey   string
	CreatedByKey   string
	UpdatedAtKey   string
	UpdatedByKey   string
}

func (t ToolModel) Name() string {
	return "tools"
}

func GetToolModel() ToolModel {
	return ToolModel{
		IdKey:          "id",
		NameKey:        "name",
		TypeKey:        "type",
		ProviderKey:    "provider",
		DescriptionKey: "description",
		EnabledKey:     "enabled",
		ToolsKey:       "tools",
		TagsKey:        "tags",
		ConfigKey:      "config",
		AuthKey:        "auth",
		CreatedAtKey:   "created_at",
		CreatedByKey:   "created_by",
		UpdatedAtKey:   "updated_at",
		UpdatedByKey:   "updated_by",
	}
}
