package health

import (
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, path string) {
	HealthCheckRoute(router, path)
	router.Get(path, HealthCheck)
}

var routeTags = []string{"health"}
