// Package config loads application configuration from environment variables.
package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	APIMode  string `env:"API_MODE" envDefault:"panel"` // "panel" or "agent"
	Port     string `env:"PORT" envDefault:"8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// Database / cache (panel only)
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://vpn:vpn@localhost:5432/vpn?sslmode=disable"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	// Auth (panel only)
	JWTSecret string `env:"JWT_SECRET" envDefault:"change-me-in-production"`

	// OAuth (panel only)
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID" envDefault:""`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET" envDefault:""`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL" envDefault:""`

	// Telegram Login (panel only)
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN" envDefault:""`
	TelegramBotName  string `env:"TELEGRAM_BOT_NAME" envDefault:""`

	// Agent-specific fields
	PanelURL     string `env:"PANEL_URL" envDefault:"http://localhost:8080"`
	NodeToken    string `env:"NODE_TOKEN" envDefault:""`
	NodePublicIP string `env:"NODE_PUBLIC_IP" envDefault:""`

	// Payments (panel only)
	PaymentProvider string `env:"PAYMENT_PROVIDER" envDefault:"stub"` // "stub" or "platega"
	PlategaAPIKey   string `env:"PLATEGA_API_KEY" envDefault:""`
	PlategaEndpoint string `env:"PLATEGA_ENDPOINT" envDefault:"https://api.platega.io/v1"`
}

// Load parses environment variables into Config.
func Load() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}
	return &cfg
}
