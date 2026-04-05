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

type Services struct {
	MCP MCPService
}

func InjectServices() *Services {
	logrus.Info("Injecting services provider")

	return &Services{
		MCP: appservices.NewMCPService(),
	}
}
