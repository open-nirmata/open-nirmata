package llmproviders

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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakeLLMModelsService struct {
	result      []dto.LLMModelItem
	err         error
	lastRequest *dto.ListLLMProviderModelsRequest
	lastTimeout time.Duration
}

func (f *fakeLLMModelsService) ListModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest, timeout time.Duration) ([]dto.LLMModelItem, error) {
	if req != nil {
		cloned := *req
		f.lastRequest = &cloned
	}
	f.lastTimeout = timeout
	return f.result, f.err
}

type fakeDB struct {
	provider   models.LLMProvider
	findOneErr error
}

func (f *fakeDB) FindOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	if f.findOneErr != nil {
		return mongo.NewSingleResultFromDocument(nil, f.findOneErr, nil)
	}
	return mongo.NewSingleResultFromDocument(f.provider, nil, nil)
}

func (f *fakeDB) Find(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeDB) InsertOne(ctx context.Context, col db.DbCollection, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return nil, nil
}

func (f *fakeDB) UpdateOne(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakeDB) DeleteOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return nil, nil
}

func (f *fakeDB) Aggregate(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeDB) CountDocuments(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}

func (f *fakeDB) BulkWrite(ctx context.Context, col db.DbCollection, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return nil, nil
}

func (f *fakeDB) UpdateMany(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakeDB) SetDbInContext(ctx context.Context) context.Context {
	return ctx
}

func (f *fakeDB) Disconnect(ctx context.Context) error {
	return nil
}

func (f *fakeDB) GetCollection(collectionName string) *mongo.Collection {
	return nil
}

func TestListLLMProviderModelsSuccess(t *testing.T) {
	service := &fakeLLMModelsService{result: []dto.LLMModelItem{{
		ID:            "gpt-4.1",
		Name:          "GPT-4.1",
		Provider:      "openai",
		Description:   "Flagship model",
		ContextWindow: 128000,
		Capabilities:  []string{"chat", "vision"},
	}}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{LLMModels: service}}))
	app.Post("/llm-providers/models", ListLLMProviderModels)

	request := httptest.NewRequest(http.MethodPost, "/llm-providers/models", strings.NewReader(`{"provider":" OpenAI ","api_key":"test-key","timeout_seconds":7}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}

	payload := dto.ListLLMProviderModelsResponse{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("expected response to decode, got: %v", err)
	}

	if !payload.Success || payload.Count != 1 || len(payload.Data) != 1 {
		t.Fatalf("unexpected response payload: %#v", payload)
	}
	if service.lastRequest == nil || service.lastRequest.Provider != "openai" {
		t.Fatalf("expected normalized provider to be passed to service, got %#v", service.lastRequest)
	}
	if service.lastTimeout != 7*time.Second {
		t.Fatalf("expected timeout to be passed through, got %v", service.lastTimeout)
	}
}

func TestListLLMProviderModelsUsesSavedProviderAndOverrides(t *testing.T) {
	service := &fakeLLMModelsService{result: []dto.LLMModelItem{{ID: "gpt-4.1", Name: "GPT-4.1", Provider: "openai"}}}
	database := &fakeDB{provider: models.LLMProvider{
		Id:           "provider-123",
		Provider:     "openai",
		BaseURL:      "https://saved.example.com",
		Organization: "saved-org",
		ProjectID:    "saved-project",
		Auth:         map[string]interface{}{"api_key": "saved-key"},
	}}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{D: database, S: &providers.Services{LLMModels: service}}))
	app.Post("/llm-providers/models", ListLLMProviderModels)

	request := httptest.NewRequest(http.MethodPost, "/llm-providers/models", strings.NewReader(`{"llm_provider_id":"provider-123","project_id":"override-project"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to succeed, got error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", response.StatusCode)
	}
	if service.lastRequest == nil {
		t.Fatalf("expected service to receive merged request")
	}
	if service.lastRequest.Provider != "openai" {
		t.Fatalf("expected provider to come from saved config, got %q", service.lastRequest.Provider)
	}
	if service.lastRequest.APIKey != "saved-key" {
		t.Fatalf("expected api key to come from saved config, got %q", service.lastRequest.APIKey)
	}
	if service.lastRequest.BaseURL != "https://saved.example.com" {
		t.Fatalf("expected base_url to come from saved config, got %q", service.lastRequest.BaseURL)
	}
	if service.lastRequest.Organization != "saved-org" {
		t.Fatalf("expected organization to come from saved config, got %q", service.lastRequest.Organization)
	}
	if service.lastRequest.ProjectID != "override-project" {
		t.Fatalf("expected request override to win for project_id, got %q", service.lastRequest.ProjectID)
	}
}

func TestListLLMProviderModelsValidation(t *testing.T) {
	service := &fakeLLMModelsService{err: errors.New("should not be called")}

	app := fiber.New()
	app.Use(providers.Handle(&providers.Provider{S: &providers.Services{LLMModels: service}}))
	app.Post("/llm-providers/models", ListLLMProviderModels)

	request := httptest.NewRequest(http.MethodPost, "/llm-providers/models", strings.NewReader(`{"provider":"unknown"}`))
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("expected request to return validation error, got: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", response.StatusCode)
	}
	if service.lastRequest != nil {
		t.Fatalf("expected service not to be called on invalid input")
	}
}
