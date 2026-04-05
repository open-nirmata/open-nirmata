package routes

import (
	"open-nirmata/config"
	"open-nirmata/providers"
	"open-nirmata/routes/health"
	"open-nirmata/routes/knowledgebases"
	"open-nirmata/routes/llmproviders"
	"open-nirmata/routes/tools"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app fiber.Router, prv *providers.Provider, cnf config.Config) {
	// apiRoutePath := "/api"
	health.RegisterRoutes(app, "/health")
	tools.RegisterRoutes(app, "/tools")
	knowledgebases.RegisterRoutes(app, "/knowledgebases")
	llmproviders.RegisterRoutes(app, "/llm-providers")
}
