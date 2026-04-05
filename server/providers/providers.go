package providers

import (
	"open-nirmata/config"
	"open-nirmata/db"

	"github.com/gofiber/fiber/v2"
)

type Provider struct {
	S *Services
	D db.DB // Database connection interface
}

func InjectDefaultProviders(cnf config.Config) (*Provider, error) {
	d, err := NewDBConnection(cnf)
	if err != nil {
		return nil, err
	}

	return &Provider{
		S: InjectServices(),
		D: d, // Initialize the database connection
	}, nil
}

type keyType struct {
	key string
}

var providerKey = keyType{"providers"}

func Handle(p *Provider) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Locals(providerKey, p)
		return c.Next()
	}
}

func GetProviders(c *fiber.Ctx) *Provider {
	return c.Locals(providerKey).(*Provider)
}
