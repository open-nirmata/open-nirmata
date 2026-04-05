package main

import (
	"context"
	"open-nirmata/config"
	"open-nirmata/metrics"
	"open-nirmata/providers"
	"open-nirmata/routes"
	"open-nirmata/utils/docs"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

var globalProvider *providers.Provider

func main() {
	version := "latest"
	if len(os.Args) > 1 {
		version = os.Args[1]
	}
	app := fiber.New(fiber.Config{
		ReadBufferSize: 8192,
		BodyLimit:      10 * 1024 * 1024 * 1024,
	})

	cnf, metricsProvider := SetupServer(app, version)

	log.Info("server setup done")

	for _, route := range app.GetRoutes() {
		if route.Method == "OPTIONS" || route.Method == "HEAD" || route.Method == "TRACE" || route.Method == "CONNECT" {
			continue
		}
		if route.Path == "/" {
			continue
		}
		log.Infof("%s %s", route.Method, route.Path)
	}

	// Graceful shutdown
	defer func() {

		if metricsProvider != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := metricsProvider.Shutdown(ctx); err != nil {
				log.Errorf("Error shutting down metrics provider: %v", err)
			}
		}
	}()

	err := app.Listen(":" + cnf.Server.Port)
	if err != nil {
		log.Fatal(err)
		return
	}
}

// GetProvider retrieves the provider from the Fiber app context
func GetProvider(app *fiber.App) *providers.Provider {
	// We need to get it from the stored context during setup
	// This is a helper function for shutdown
	return globalProvider
}

func SetupServer(app *fiber.App, version string) (*config.Config, *metrics.MetricsProvider) {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	cnf := config.NewConfig()
	log.SetLevel(cnf.Server.LogLevel)
	cnf.Deployment.Version = version
	log.WithFields(log.Fields{
		"host":     cnf.Server.Host,
		"port":     cnf.Server.Port,
		"env":      cnf.Deployment.Environment,
		"app_name": cnf.Deployment.Name,
	}).Info("Loaded Configurations")
	// Initialize metrics provider
	metricsProvider, err := metrics.NewMetricsProvider(cnf.Deployment.Name, "1.0.0")
	if err != nil {
		log.Errorf("Failed to initialize metrics provider: %v", err)
		// Continue without metrics
	}
	if !cnf.Server.EnableMetrics {
		log.Info("Metrics collection is not enabled")
		metricsProvider = nil
	} else {
		log.Info("Metrics collection is enabled")
	}

	prv, err := providers.InjectDefaultProviders(*cnf)
	if err != nil {
		log.Fatalf("error injecting providers %s", err)
	}

	// Store provider globally for graceful shutdown
	globalProvider = prv

	log.Info("providers loaded. setting up middleware...")

	// Add metrics middleware for HTTP request tracking
	if metricsProvider != nil {
		app.Use(metricsProvider.FiberMiddleware())
	}

	app.Use((*cnf).Handle)
	app.Use(providers.Handle(prv))
	// if prv.Settings.AllowedOrigins is not empty, use it for CORS configuration, otherwise allow all origins

	app.Use(cors.New())

	// Add metrics endpoint
	if metricsProvider != nil {
		app.Get("/metrics", metricsProvider.MetricsHandler())
		log.Info("Metrics endpoint available at /metrics")
	}

	app.Static("", "static", fiber.Static{
		Index: "index.html",
	})
	app.Static("/docs", "./docs")
	routes.RegisterRoutes(app, prv, *cnf)

	if cnf.EnableDocsCreation {
		err = docs.CreateOpenApiDoc("docs/open-nirmata.yaml")
		if err != nil {
			log.Fatal("failed to create OpenAPI doc: %w", err)
		}
	}

	return cnf, metricsProvider
}
