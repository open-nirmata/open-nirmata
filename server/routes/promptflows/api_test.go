package promptflows

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

type fakePromptFlowDB struct {
	insertedFlow   models.PromptFlow
	duplicateCount int64
	nameCounts     map[string]int64
	flowsByID      map[string]models.PromptFlow
	llmProviders   map[string]models.LLMProvider
	tools          map[string]models.Tool
	knowledgebases map[string]models.Knowledgebase
}

func (f *fakePromptFlowDB) FindOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	id := ""
	if asMap, ok := filter.(bson.M); ok {
		if value, ok := asMap["id"].(string); ok {
			id = value
		}
	}

	switch col.Name() {
	case "llm_providers":
		if record, ok := f.llmProviders[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
	case "tools":
		if record, ok := f.tools[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
	case "knowledgebases":
		if record, ok := f.knowledgebases[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
	case "prompt_flows":
		if record, ok := f.flowsByID[id]; ok {
			return mongo.NewSingleResultFromDocument(record, nil, nil)
		}
		if f.insertedFlow.Id != "" && f.insertedFlow.Id == id {
			return mongo.NewSingleResultFromDocument(f.insertedFlow, nil, nil)
		}
	}

	return mongo.NewSingleResultFromDocument(nil, mongo.ErrNoDocuments, nil)
}

func (f *fakePromptFlowDB) Find(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakePromptFlowDB) InsertOne(ctx context.Context, col db.DbCollection, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	if flow, ok := document.(models.PromptFlow); ok {
		f.insertedFlow = flow
		if f.flowsByID == nil {
			f.flowsByID = map[string]models.PromptFlow{}
		}
		f.flowsByID[flow.Id] = flow
	}
	return &mongo.InsertOneResult{InsertedID: "created"}, nil
}

func (f *fakePromptFlowDB) UpdateOne(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

func (f *fakePromptFlowDB) DeleteOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return &mongo.DeleteResult{DeletedCount: 1}, nil
}

func (f *fakePromptFlowDB) Aggregate(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakePromptFlowDB) CountDocuments(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	if asMap, ok := filter.(bson.M); ok {
		if name, ok := asMap["name"].(string); ok {
			if count, exists := f.nameCounts[name]; exists {
				return count, nil
			}
		}
	}
	return f.duplicateCount, nil
}

func (f *fakePromptFlowDB) BulkWrite(ctx context.Context, col db.DbCollection, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return nil, nil
}

func (f *fakePromptFlowDB) UpdateMany(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakePromptFlowDB) SetDbInContext(ctx context.Context) context.Context {
	return ctx
}

func (f *fakePromptFlowDB) Disconnect(ctx context.Context) error {
	return nil
}

func (f *fakePromptFlowDB) GetCollection(collectionName string) *mongo.Collection {
	return nil
}

func TestValidatePromptFlowSuccess(t *testing.T) {
	database := &fakePromptFlowDB{
		llmProviders: map[string]models.LLMProvider{
			"provider-1": {Id: "provider-1", Name: "Primary", Provider: "openai", Enabled: true},
		},
		tools: map[string]models.Tool{
			"tool-1": {Id: "tool-1", Name: "Search", Type: "http", Enabled: true},
		},
		knowledgebases: map[string]models.Knowledgebase{
			"kb-1": {Id: "kb-1", Name: "Docs", Provider: "qdrant", Enabled: true},
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/prompt-flows/validate", ValidatePromptFlow)

	requestBody := `{
		"name":"Support Agent",
		"defaults":{
			"llm_provider_id":"provider-1",
			"model":"gpt-4.1",
			"tool_ids":["tool-1"],
			"knowledgebase_ids":["kb-1"]
		},
		"stages":[
			{"id":"triage","name":"Triage","type":"router","transitions":[{"label":"billing","target_stage_id":"billing"},{"label":"product","target_stage_id":"product"}]},
			{"id":"billing","name":"Billing","type":"llm","prompt":"Help with billing","overrides":{"tool_ids":[]},"on_success":"product"},
			{"id":"product","name":"Product","type":"result"}
		]
	}`

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows/validate", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}

	payload := dto.PromptFlowValidateResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil || !payload.Data.Valid {
		t.Fatalf("unexpected validation payload: %#v", payload)
	}
	if payload.Data.EntryStageID != "triage" {
		t.Fatalf("expected entry stage to resolve to triage, got %q", payload.Data.EntryStageID)
	}
	if len(payload.Data.Stages) != 3 {
		t.Fatalf("expected 3 resolved stages, got %d", len(payload.Data.Stages))
	}

	for _, stage := range payload.Data.Stages {
		if stage.Id == "billing" {
			if stage.Effective == nil {
				t.Fatalf("expected effective resources for billing stage")
			}
			if len(stage.Effective.ToolIDs) != 0 {
				t.Fatalf("expected billing override to clear inherited tools, got %#v", stage.Effective.ToolIDs)
			}
		}
	}
}

func TestValidatePromptFlowRejectsUnknownStageTransition(t *testing.T) {
	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: &fakePromptFlowDB{}}))
	app.Post("/prompt-flows/validate", ValidatePromptFlow)

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows/validate", strings.NewReader(`{"name":"Broken Flow","stages":[{"id":"start","name":"Start","type":"router","transitions":[{"target_stage_id":"missing"}]}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to return validation error, got: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", response.StatusCode)
	}
}

func TestCreatePromptFlowSuccess(t *testing.T) {
	database := &fakePromptFlowDB{
		llmProviders: map[string]models.LLMProvider{
			"provider-1": {Id: "provider-1", Name: "Primary", Provider: "openai", Enabled: true},
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/prompt-flows", CreatePromptFlow)

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows", strings.NewReader(`{"name":"Support Flow","include_conversation_history":false,"defaults":{"llm_provider_id":"provider-1","model":"gpt-4.1"},"stages":[{"id":"start","name":"Start","type":"llm","prompt":"Be helpful","on_success":"done"},{"id":"done","name":"Done","type":"result"}]}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", response.StatusCode)
	}

	payload := dto.PromptFlowResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if payload.Data.EntryStageID != "start" {
		t.Fatalf("expected entry_stage_id to default to first stage, got %q", payload.Data.EntryStageID)
	}
	if payload.Data.IncludeConversationHistory == nil || *payload.Data.IncludeConversationHistory {
		t.Fatalf("expected include_conversation_history=false in response, got %#v", payload.Data.IncludeConversationHistory)
	}
	if database.insertedFlow.Name != "Support Flow" {
		t.Fatalf("expected flow to be inserted, got %#v", database.insertedFlow)
	}
	if database.insertedFlow.IncludeConversationHistory == nil || *database.insertedFlow.IncludeConversationHistory {
		t.Fatalf("expected include_conversation_history=false to be stored, got %#v", database.insertedFlow.IncludeConversationHistory)
	}
}

func TestCopyPromptFlowSuccess(t *testing.T) {
	enabled := false
	source := models.PromptFlow{
		Id:                         "flow-1",
		Name:                       "Support Flow",
		Description:                "Original description",
		Enabled:                    enabled,
		IncludeConversationHistory: &enabled,
		Defaults: &models.PromptFlowResources{
			LLMProviderID: "provider-1",
			Model:         "gpt-4.1",
			ToolIDs:       []string{"tool-1"},
		},
		EntryStageID: "start",
		Stages: []models.PromptFlowStage{
			{Id: "start", Name: "Start", Type: dto.PromptFlowStageTypeLLM, Prompt: "Help the user", OnSuccess: "done", Overrides: &models.PromptFlowResources{KnowledgebaseIDs: []string{"kb-1"}}},
			{Id: "done", Name: "Done", Type: dto.PromptFlowStageTypeResult},
		},
	}

	database := &fakePromptFlowDB{
		flowsByID: map[string]models.PromptFlow{
			source.Id: source,
		},
		llmProviders: map[string]models.LLMProvider{
			"provider-1": {Id: "provider-1", Name: "Primary", Provider: "openai", Enabled: true},
		},
		tools: map[string]models.Tool{
			"tool-1": {Id: "tool-1", Name: "Search", Type: "http", Enabled: true},
		},
		knowledgebases: map[string]models.Knowledgebase{
			"kb-1": {Id: "kb-1", Name: "Docs", Provider: "qdrant", Enabled: true},
		},
		nameCounts: map[string]int64{
			"Support Flow Copy": 0,
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/prompt-flows/:id/copy", CopyPromptFlow)

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows/flow-1/copy", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", response.StatusCode)
	}

	payload := dto.PromptFlowResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if !payload.Success || payload.Data == nil {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if payload.Data.Id == "" || payload.Data.Id == source.Id {
		t.Fatalf("expected copied flow to have a new id, got %#v", payload.Data)
	}
	if payload.Data.Name != "Support Flow Copy" {
		t.Fatalf("expected auto-generated copy name, got %q", payload.Data.Name)
	}
	if payload.Data.Description != source.Description {
		t.Fatalf("expected description to be copied, got %q", payload.Data.Description)
	}
	if payload.Data.Enabled != source.Enabled {
		t.Fatalf("expected enabled state to be preserved, got %v", payload.Data.Enabled)
	}
	if len(payload.Data.Stages) != len(source.Stages) {
		t.Fatalf("expected stages to be copied, got %#v", payload.Data.Stages)
	}
	if database.insertedFlow.Name != "Support Flow Copy" {
		t.Fatalf("expected copied flow to be inserted with generated name, got %#v", database.insertedFlow)
	}
	if database.insertedFlow.Id == source.Id {
		t.Fatalf("expected inserted copied flow to have a new id, got %#v", database.insertedFlow)
	}
	if database.insertedFlow.Defaults == source.Defaults || database.insertedFlow.Stages[0].Overrides == source.Stages[0].Overrides {
		t.Fatalf("expected copied flow resources to be deep-cloned")
	}
}

func TestCopyPromptFlowAutoGeneratesNextAvailableName(t *testing.T) {
	source := models.PromptFlow{
		Id:           "flow-1",
		Name:         "Support Flow",
		Enabled:      true,
		EntryStageID: "done",
		Stages: []models.PromptFlowStage{
			{Id: "done", Name: "Done", Type: dto.PromptFlowStageTypeResult},
		},
	}

	database := &fakePromptFlowDB{
		flowsByID: map[string]models.PromptFlow{
			source.Id: source,
		},
		nameCounts: map[string]int64{
			"Support Flow Copy":   1,
			"Support Flow Copy 2": 0,
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/prompt-flows/:id/copy", CopyPromptFlow)

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows/flow-1/copy", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 response, got %d", response.StatusCode)
	}

	payload := dto.PromptFlowResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response body to decode, got error: %v", err)
	}
	if payload.Data == nil || payload.Data.Name != "Support Flow Copy 2" {
		t.Fatalf("expected next available copy name, got %#v", payload.Data)
	}
}

func TestCopyPromptFlowRejectsDuplicateExplicitName(t *testing.T) {
	source := models.PromptFlow{
		Id:           "flow-1",
		Name:         "Support Flow",
		Enabled:      true,
		EntryStageID: "done",
		Stages: []models.PromptFlowStage{
			{Id: "done", Name: "Done", Type: dto.PromptFlowStageTypeResult},
		},
	}

	database := &fakePromptFlowDB{
		flowsByID: map[string]models.PromptFlow{
			source.Id: source,
		},
		nameCounts: map[string]int64{
			"Existing Copy": 1,
		},
	}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database}))
	app.Post("/prompt-flows/:id/copy", CopyPromptFlow)

	request := httptest.NewRequest(http.MethodPost, "/prompt-flows/flow-1/copy", strings.NewReader(`{"name":"Existing Copy"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to return validation error, got: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", response.StatusCode)
	}
}
