package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"open-nirmata/dto"
)

func TestLLMModelsServiceListModelsOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/models" {
			t.Fatalf("expected /v1/models path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1","owned_by":"openai","description":"General model","context_window":128000}]}`))
	}))
	defer server.Close()

	service := NewLLMModelsService()
	models, err := service.ListModels(context.Background(), &dto.ListLLMProviderModelsRequest{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("expected list models to succeed, got %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one model, got %d", len(models))
	}
	if models[0].ID != "gpt-4.1" || models[0].Provider != "openai" {
		t.Fatalf("unexpected model payload: %#v", models[0])
	}
	if models[0].ContextWindow != 128000 {
		t.Fatalf("expected context window 128000, got %d", models[0].ContextWindow)
	}
}

func TestLLMModelsServiceListModelsOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("expected /api/tags path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest","size":2048,"details":{"family":"llama","parameter_size":"3B"}}]}`))
	}))
	defer server.Close()

	service := NewLLMModelsService()
	models, err := service.ListModels(context.Background(), &dto.ListLLMProviderModelsRequest{
		Provider: "ollama",
		BaseURL:  server.URL,
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("expected ollama list models to succeed, got %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one model, got %d", len(models))
	}
	if models[0].ID != "llama3.2:latest" || models[0].Provider != "ollama" {
		t.Fatalf("unexpected ollama model payload: %#v", models[0])
	}
	if len(models[0].Capabilities) == 0 || models[0].Capabilities[0] != "llama" {
		t.Fatalf("expected ollama capabilities to include family, got %#v", models[0].Capabilities)
	}
}

func TestLLMModelsServiceValidation(t *testing.T) {
	service := NewLLMModelsService()

	_, err := service.ListModels(context.Background(), &dto.ListLLMProviderModelsRequest{Provider: "anthropic"}, 5*time.Second)
	if err == nil {
		t.Fatalf("expected validation error when api key is missing")
	}
}
