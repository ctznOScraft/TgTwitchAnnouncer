package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	TelegramToken      string // Token for the Telegram Bot API.
	TwitchClientID     string // Client ID for Twitch API authentication.
	TwitchClientSecret string // Client Secret for Twitch API authentication.
	CallbackUrl        string // Publicly accessible URL for Twitch EventSub webhooks (e.g., https://yourdomain.com/webhook).
	WebhookSecret      string // Secret string used to verify incoming Twitch EventSub payloads.
	Port               string // Port on which the local HTTP server will listen (defaults to 4192).
}

// LoadConfig reads configuration from the environment and a .env file if available.
// It returns an error if any of the required fields are missing.
func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // Ignore error, as env vars might be provided directly via host/Docker without a .env file

	cfg := Config{
		TelegramToken:      os.Getenv("TG_TOKEN"),
		TwitchClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		TwitchClientSecret: os.Getenv("TWITCH_SECRET"),
		CallbackUrl:        os.Getenv("CALLBACK_URL"),
		WebhookSecret:      os.Getenv("WEBHOOK_SECRET"),
		Port:               os.Getenv("PORT"),
	}

	if cfg.TelegramToken == "" {
		return nil, errors.New("error: .env TG_TOKEN required")
	}
	if cfg.TwitchClientID == "" {
		return nil, errors.New("error: .env TWITCH_CLIENT_ID required")
	}
	if cfg.TwitchClientSecret == "" {
		return nil, errors.New("error: .env TWITCH_SECRET required")
	}
	if cfg.CallbackUrl == "" {
		return nil, errors.New("error: .env CALLBACK_URL required")
	}
	if cfg.WebhookSecret == "" {
		return nil, errors.New("error: .env WEBHOOK_SECRET required")
	}
	if cfg.Port == "" {
		cfg.Port = "4192"
	}

	return &cfg, nil
}
