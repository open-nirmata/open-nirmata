package providers

import (
	"context"
	"time"

	"open-nirmata/dto"
	appservices "open-nirmata/services"

	"github.com/sirupsen/logrus"
)

type MCPService interface {
	ListTools(ctx context.Context, config *dto.ToolConfig, timeout time.Duration) (*dto.TestMCPToolResult, error)
}

type LLMModelsService interface {
	ListModels(ctx context.Context, req *dto.ListLLMProviderModelsRequest, timeout time.Duration) ([]dto.LLMModelItem, error)
}

type ChatCompletionService interface {
	ChatCompletion(ctx context.Context, req *appservices.ChatCompletionRequest, timeout time.Duration) (*appservices.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *appservices.ChatCompletionRequest, timeout time.Duration, callback func(*appservices.ChatCompletionStreamChunk) error) (*appservices.ChatCompletionResponse, error)
}

type ToolExecutorService interface {
	ExecuteTool(ctx context.Context, req *appservices.ToolExecutionRequest, timeout time.Duration) (*appservices.ToolExecutionResult, error)
	ExecuteTools(ctx context.Context, requests []*appservices.ToolExecutionRequest, timeout time.Duration) ([]*appservices.ToolExecutionResult, error)
}

type KnowledgeRetrieverService interface {
	RetrieveContext(ctx context.Context, req *appservices.RetrievalRequest, timeout time.Duration) ([]appservices.RetrievalResult, error)
}

type Services struct {
	MCP                MCPService
	LLMModels          LLMModelsService
	ChatCompletion     ChatCompletionService
	ToolExecutor       ToolExecutorService
	KnowledgeRetriever KnowledgeRetrieverService
}

func InjectServices() *Services {
	logrus.Info("Injecting services provider")

	return &Services{
		MCP:                appservices.NewMCPService(),
		LLMModels:          appservices.NewLLMModelsService(),
		ChatCompletion:     appservices.NewChatCompletionService(),
		ToolExecutor:       appservices.NewToolExecutorService(),
		KnowledgeRetriever: appservices.NewKnowledgeRetrieverService(),
	}
}
