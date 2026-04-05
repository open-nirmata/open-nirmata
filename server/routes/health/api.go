package health

import (
	"net/http"
	"open-nirmata/config"
	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

func HealthCheckRoute(router fiber.Router, path string) {
	router.Get(path, HealthCheck)
	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "Health Check",
		Description: "Health Check",
		Response: &docs.ApiResponse{
			Description: "Ingestion is successful",
			Content:     new(dto.HealthCheckResponse),
		},
		Tags: routeTags,
	})
}

func HealthCheck(c *fiber.Ctx) error {
	cnf := config.GetAppConfig(c)
	return c.JSON(dto.HealthCheckResponse{
		Success: true,
		Version: cnf.Deployment.Version,
	})
}
