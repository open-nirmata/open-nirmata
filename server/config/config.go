package config

import (
	"open-nirmata/dto"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Server                ServerConfig     `json:"server"`
	Deployment            DeploymentConfig `json:"deployment"`
	EnableDocsCreation    bool             `json:"enable_docs_creation"`
	EnableAuth            bool             `json:"enable_auth"`
	DB                    DB               `json:"db"`
	Auth                  GoIAM            `json:"auth"`
	SettingsEncryptionKey dto.MaskedBytes  `json:"-"`
}

type ServerConfig struct {
	Host                string       `json:"host"`
	Port                string       `json:"port"`
	EnableMetrics       bool         `json:"enable_metrics"`
	AutoLoadCollections []string     `json:"auto_load_collections"`
	LogLevel            logrus.Level `json:"log_level"`
}

type DeploymentConfig struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	ID          string `json:"id"`
}

type GoIAM struct {
	BaseURL      string          `json:"base_url"`
	ClientID     string          `json:"client_id"`
	clientSecret dto.MaskedBytes `json:"-"`
}

func (g GoIAM) Secret() string {
	return string(g.clientSecret)
}

func NewConfig() *Config {
	godotenv.Load()
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "3000"
	}
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	c := &Config{
		SettingsEncryptionKey: dto.MaskedBytes(os.Getenv("SETTINGS_ENCRYPTION_KEY")),
		Server: ServerConfig{
			Host:          host,
			Port:          port,
			EnableMetrics: os.Getenv("ENABLE_METRICS") == "true",
			LogLevel:      logrus.InfoLevel,
		},
		Deployment: DeploymentConfig{
			Environment: "development",
			Name:        "open-nirmata",
		},
		DB: DB{
			host: os.Getenv("DB_HOST"),
		},
		EnableDocsCreation: os.Getenv("ENABLE_DOCS_CREATION") == "true",
		EnableAuth:         os.Getenv("ENABLE_AUTH") == "true",
	}
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err == nil {
		c.Server.LogLevel = level
	}
	return c
}

type keyType struct {
	key string
}

var configKey = keyType{"config"}

func (a *Config) Handle(c *fiber.Ctx) error {
	c.Locals(configKey, *a)
	return c.Next()
}

func GetAppConfig(c *fiber.Ctx) Config {
	return c.Locals(configKey).(Config)
}

type Jwt struct {
	secret dto.MaskedBytes // JWT secret key (private field, use Secret() method to access)
}

// Secret returns the JWT secret key used for signing and verifying JWT tokens.
// The secret is stored as MaskedBytes for security purposes.
//
// Returns the JWT secret configured via JWT_SECRET environment variable.
func (j Jwt) Secret() dto.MaskedBytes {
	return j.secret
}

// DB holds database configuration settings.
type DB struct {
	host string // MongoDB connection string (private field)
}

// Host returns the database connection string.
// This is the primary method to access the MongoDB connection URL.
//
// Returns the MongoDB connection string configured via DB_HOST environment variable.
func (d DB) Host() string {
	return d.host
}
