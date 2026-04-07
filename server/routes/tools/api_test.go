package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"open-nirmata/db"
	"open-nirmata/db/models"
	"open-nirmata/dto"
	"open-nirmata/providers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakeMCPService struct {
	result      *dto.TestMCPToolResult
	err         error
	lastConfig  *dto.ToolConfig
	lastTimeout time.Duration
}

type fakeToolDB struct {
	tool         models.Tool
	insertedTool *models.Tool
	updatedTool  *models.Tool
	findOneErr   error
	insertErr    error
	updateErr    error
}

func (f *fakeMCPService) ListTools(ctx context.Context, config *dto.ToolConfig, timeout time.Duration) (*dto.TestMCPToolResult, error) {
	f.lastConfig = config
	f.lastTimeout = timeout
	return f.result, f.err
}

func (f *fakeToolDB) FindOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	if f.findOneErr != nil {
		return mongo.NewSingleResultFromDocument(nil, f.findOneErr, nil)
	}
	return mongo.NewSingleResultFromDocument(f.tool, nil, nil)
}

func (f *fakeToolDB) Find(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeToolDB) InsertOne(ctx context.Context, col db.DbCollection, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	tool, _ := document.(models.Tool)
	toolCopy := tool
	f.insertedTool = &toolCopy
	f.tool = toolCopy
	return &mongo.InsertOneResult{InsertedID: tool.Id}, nil
}

func (f *fakeToolDB) UpdateOne(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	if updateDoc, ok := update.(bson.M); ok {
		if setDoc, ok := updateDoc["$set"].(bson.M); ok {
			updated := f.tool
			if value, ok := setDoc["name"].(string); ok {
				updated.Name = value
			}
			if value, ok := setDoc["type"].(string); ok {
				updated.Type = value
			}
			if value, ok := setDoc["provider"].(string); ok {
				updated.Provider = value
			}
			if value, ok := setDoc["description"].(string); ok {
				updated.Description = value
			}
			if value, ok := setDoc["enabled"].(bool); ok {
				updated.Enabled = value
			}
			if value, ok := setDoc["tags"].([]string); ok {
				updated.Tags = value
			}
			if value, ok := setDoc["config"].(*dto.ToolConfig); ok {
				updated.Config = value
			}
			if value, ok := setDoc["auth"].(map[string]interface{}); ok {
				updated.Auth = value
			}
			if value, ok := setDoc["updated_by"].(string); ok {
				updated.UpdatedBy = value
			}
			if value, ok := setDoc["updated_at"].(*time.Time); ok {
				updated.UpdatedAt = value
			}
			f.tool = updated
			updatedCopy := updated
			f.updatedTool = &updatedCopy
		}
	}

	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func (f *fakeToolDB) DeleteOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return &mongo.DeleteResult{DeletedCount: 1}, nil
}

func (f *fakeToolDB) Aggregate(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeToolDB) CountDocuments(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}

func (f *fakeToolDB) BulkWrite(ctx context.Context, col db.DbCollection, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return nil, nil
}

func (f *fakeToolDB) UpdateMany(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakeToolDB) SetDbInContext(ctx context.Context) context.Context {
	return ctx
}

func (f *fakeToolDB) Disconnect(ctx context.Context) error {
	return nil
}

func (f *fakeToolDB) GetCollection(collectionName string) *mongo.Collection {
	return nil
}

func TestTestMCPToolHandlerSuccess(t *testing.T) {
	service := &fakeMCPService{result: &dto.TestMCPToolResult{
		Transport: "stdio",
		Tools: []dto.MCPDiscoveredTool{{
			Name:        "filesystem",
			Description: "Lists files",
		}},
		Count: 1,
	}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{MCP: service}}))
	app.Post("/tools/test", TestMCPTool)

	request := httptest.NewRequest(http.MethodPost, "/tools/test", strings.NewReader(`{"config":{"transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"]},"timeout_seconds":5}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}

	payload := dto.TestMCPToolResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil || payload.Data.Count != 1 {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if service.lastTimeout != 5*time.Second {
		t.Fatalf("expected timeout to be passed through, got %v", service.lastTimeout)
	}
}

func TestTestMCPToolHandlerValidation(t *testing.T) {
	service := &fakeMCPService{err: errors.New("should not be called")}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{MCP: service}}))
	app.Post("/tools/test", TestMCPTool)

	request := httptest.NewRequest(http.MethodPost, "/tools/test", strings.NewReader(`{"config":{"transport":"stdio"}}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to return validation error, got: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", response.StatusCode)
	}
	if service.lastConfig != nil {
		t.Fatalf("expected service not to be called on invalid input")
	}
}

func TestCreateToolRefreshesMCPMetadata(t *testing.T) {
	service := &fakeMCPService{result: &dto.TestMCPToolResult{
		Transport: "stdio",
		ServerInfo: &dto.MCPServerInfo{
			Name:    "filesystem-server",
			Version: "1.0.0",
		},
		Tools: []dto.MCPDiscoveredTool{{
			Name:        "filesystem",
			Description: "Lists files from disk",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
			},
			Annotations: map[string]interface{}{"category": "fs"},
		}},
		Count: 1,
	}}
	database := &fakeToolDB{}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database, S: &providers.Services{MCP: service}}))
	app.Post("/tools", CreateTool)

	request := httptest.NewRequest(http.MethodPost, "/tools", strings.NewReader(`{"name":"filesystem","type":"mcp","description":"stale","config":{"transport":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"]}}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected create request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", response.StatusCode)
	}
	if database.insertedTool == nil {
		t.Fatalf("expected tool to be inserted")
	}
	if database.insertedTool.Description != "Lists files from disk" {
		t.Fatalf("expected description to be refreshed from MCP metadata, got %q", database.insertedTool.Description)
	}
	if database.insertedTool.Config == nil || len(database.insertedTool.Config.InputSchema) == 0 {
		t.Fatalf("expected input schema to be stored on the created tool, got %#v", database.insertedTool.Config)
	}
	if database.insertedTool.Config.ServerInfo == nil || database.insertedTool.Config.ServerInfo.Name != "filesystem-server" {
		t.Fatalf("expected server info to be stored, got %#v", database.insertedTool.Config.ServerInfo)
	}
}

func TestUpdateToolRefreshesMCPMetadata(t *testing.T) {
	service := &fakeMCPService{result: &dto.TestMCPToolResult{
		Transport: "stdio",
		Tools: []dto.MCPDiscoveredTool{{
			Name:        "filesystem",
			Description: "Updated from MCP",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}}},
		}},
		Count: 1,
	}}
	database := &fakeToolDB{tool: models.Tool{
		Id:          "tool-1",
		Name:        "filesystem",
		Type:        string(dto.ToolTypeMCP),
		Description: "old description",
		Enabled:     true,
		Config: &dto.ToolConfig{
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
		},
	}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database, S: &providers.Services{MCP: service}}))
	app.Put("/tools/:id", UpdateTool)

	request := httptest.NewRequest(http.MethodPut, "/tools/tool-1", strings.NewReader(`{"description":"manual override"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected update request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}
	if database.updatedTool == nil {
		t.Fatalf("expected tool to be updated")
	}
	if database.updatedTool.Description != "Updated from MCP" {
		t.Fatalf("expected description to be overwritten from MCP metadata, got %q", database.updatedTool.Description)
	}
	if database.updatedTool.Config == nil || len(database.updatedTool.Config.InputSchema) == 0 {
		t.Fatalf("expected refreshed schema to be persisted, got %#v", database.updatedTool.Config)
	}
}

func TestRefreshToolHandlerSuccess(t *testing.T) {
	service := &fakeMCPService{result: &dto.TestMCPToolResult{
		Transport:  "stdio",
		ServerInfo: &dto.MCPServerInfo{Name: "filesystem-server", Version: "2.0.0"},
		Tools: []dto.MCPDiscoveredTool{{
			Name:        "filesystem",
			Description: "Refreshed tool",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}}},
			Annotations: map[string]interface{}{"category": "filesystem"},
		}},
		Count: 1,
	}}
	database := &fakeToolDB{tool: models.Tool{
		Id:          "tool-1",
		Name:        "filesystem",
		Type:        string(dto.ToolTypeMCP),
		Description: "old description",
		Enabled:     true,
		Config: &dto.ToolConfig{
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
		},
	}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database, S: &providers.Services{MCP: service}}))
	app.Post("/tools/:id/refresh", RefreshTool)

	request := httptest.NewRequest(http.MethodPost, "/tools/tool-1/refresh", strings.NewReader(`{"timeout_seconds":9}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected refresh request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}
	if service.lastTimeout != 9*time.Second {
		t.Fatalf("expected timeout override to be honored, got %v", service.lastTimeout)
	}
	if database.updatedTool == nil || database.updatedTool.Config == nil || database.updatedTool.Config.LastRefreshedAt == nil {
		t.Fatalf("expected refresh metadata to be persisted, got %#v", database.updatedTool)
	}
	if database.updatedTool.Description != "Refreshed tool" {
		t.Fatalf("expected refreshed description, got %q", database.updatedTool.Description)
	}
}
