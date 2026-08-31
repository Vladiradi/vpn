// Package config loads application configuration from environment variables.
package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

// Config holds all configuration for the application.
type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://vpn:vpn@localhost:5432/vpn?sslmode=disable"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"change-me-in-production"`

	// Agent-specific fields
	PanelURL string `env:"PANEL_URL" envDefault:"http://localhost:8080"`
	NodeToken string `env:"NODE_TOKEN" envDefault:""`
}

// Load parses environment variables into Config.
func Load() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}
	return &cfg
}
