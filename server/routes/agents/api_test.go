package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"open-nirmata/db"
	"open-nirmata/db/models"
	"open-nirmata/dto"
	"open-nirmata/providers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakeAgentDB struct {
	insertedAgent  models.Agent
	duplicateCount int64
	promptFlows    map[string]models.PromptFlow
	agents         map[string]models.Agent
}

func (f *fakeAgentDB) FindOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	id := ""
	if asMap, ok := filter.(bson.M); ok {
		if value, ok := asMap["id"].(string); ok {
			id = value
		}
	}

	switch col.Name() {
	case "prompt_flows":
		if record, ok := f.promptFlows[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
	case "agents":
		if record, ok := f.agents[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
		if f.insertedAgent.Id != "" && f.insertedAgent.Id == id {
			return mongo.NewSingleResultFromDocument(f.insertedAgent, nil, nil)
		}
	}

	return mongo.NewSingleResultFromDocument(nil, mongo.ErrNoDocuments, nil)
}

func (f *fakeAgentDB) Find(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeAgentDB) InsertOne(ctx context.Context, col db.DbCollection, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	if agent, ok := document.(models.Agent); ok {
		f.insertedAgent = agent
	}
	return &mongo.InsertOneResult{InsertedID: "created"}, nil
}

func (f *fakeAgentDB) UpdateOne(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func (f *fakeAgentDB) DeleteOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return &mongo.DeleteResult{DeletedCount: 1}, nil
}

func (f *fakeAgentDB) Aggregate(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeAgentDB) CountDocuments(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return f.duplicateCount, nil
}

func (f *fakeAgentDB) BulkWrite(ctx context.Context, col db.DbCollection, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return nil, nil
}

func (f *fakeAgentDB) UpdateMany(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakeAgentDB) SetDbInContext(ctx context.Context) context.Context {
	return ctx
}

func (f *fakeAgentDB) Disconnect(ctx context.Context) error {
	return nil
}

func (f *fakeAgentDB) GetCollection(collectionName string) *mongo.Collection {
	return nil
}

func TestValidateAgentSuccess(t *testing.T) {
	database := &fakeAgentDB{
		promptFlows: map[string]models.PromptFlow{
			"flow-1": {Id: "flow-1", Name: "Support Flow", Enabled: false},
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/agents/validate", ValidateAgent)

	request := httptest.NewRequest(http.MethodPost, "/agents/validate", strings.NewReader(`{"name":"Support Bot","type":"chat","prompt_flow_id":"flow-1"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}

	payload := dto.AgentResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success {
		t.Fatalf("unexpected validation payload: %#v", payload)
	}
	if len(payload.Warnings) != 1 {
		t.Fatalf("expected one warning for disabled prompt flow, got %#v", payload.Warnings)
	}
}

func TestCreateAgentSuccess(t *testing.T) {
	database := &fakeAgentDB{
		promptFlows: map[string]models.PromptFlow{
			"flow-1": {Id: "flow-1", Name: "Support Flow", Enabled: true},
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/agents", CreateAgent)

	request := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{"name":"Support Bot","type":"chat","prompt_flow_id":"flow-1"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", response.StatusCode)
	}

	payload := dto.AgentResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if payload.Data.Name != "Support Bot" {
		t.Fatalf("expected created agent name to be returned, got %#v", payload.Data)
	}
	if database.insertedAgent.Name != "Support Bot" {
		t.Fatalf("expected agent to be inserted, got %#v", database.insertedAgent)
	}
}

func TestBuildInitialMessagesIncludesConversationHistoryByDefault(t *testing.T) {
	req := dto.ExecuteAgentRequest{
		Messages: []dto.ExecutionMessageItem{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "latest"},
		},
	}

	messages := buildInitialMessages(req, nil)

	if len(messages) != 3 {
		t.Fatalf("expected full conversation history by default, got %d messages", len(messages))
	}
	if messages[2].Role != "user" || messages[2].Content != "latest" {
		t.Fatalf("expected latest user message to remain in position, got %#v", messages[2])
	}
}

func TestBuildInitialMessagesWithoutConversationHistoryKeepsLatestUserMessage(t *testing.T) {
	includeConversationHistory := false
	req := dto.ExecuteAgentRequest{
		Messages: []dto.ExecutionMessageItem{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "latest"},
		},
	}

	messages := buildInitialMessages(req, &includeConversationHistory)

	if len(messages) != 1 {
		t.Fatalf("expected only the latest user message when history is disabled, got %d messages", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "latest" {
		t.Fatalf("expected latest user message to be preserved, got %#v", messages[0])
	}
}
