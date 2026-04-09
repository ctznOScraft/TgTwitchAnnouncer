package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	os.Setenv("PORT", "8080")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("CALLBACK_URL")
		os.Unsetenv("WEBHOOK_SECRET")
		os.Unsetenv("PORT")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.TelegramToken != "test_token" {
		t.Errorf("expected TelegramToken='test_token', got '%s'", cfg.TelegramToken)
	}
	if cfg.TwitchClientID != "test_client_id" {
		t.Errorf("expected TwitchClientID='test_client_id', got '%s'", cfg.TwitchClientID)
	}
	if cfg.TwitchClientSecret != "test_client_secret" {
		t.Errorf("expected TwitchClientSecret='test_client_secret', got '%s'", cfg.TwitchClientSecret)
	}
	if cfg.CallbackUrl != "https://example.com/webhook" {
		t.Errorf("expected CallbackUrl='https://example.com/webhook', got '%s'", cfg.CallbackUrl)
	}
	if cfg.WebhookSecret != "test_webhook_secret" {
		t.Errorf("expected WebhookSecret='test_webhook_secret', got '%s'", cfg.WebhookSecret)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port='8080', got '%s'", cfg.Port)
	}
}

func TestLoadConfig_DefaultPort(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	os.Unsetenv("PORT")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("CALLBACK_URL")
		os.Unsetenv("WEBHOOK_SECRET")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Port != "4192" {
		t.Errorf("expected default Port='4192', got '%s'", cfg.Port)
	}
}

func TestLoadConfig_MissingToken(t *testing.T) {
	os.Unsetenv("TG_TOKEN")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	defer func() {
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("CALLBACK_URL")
		os.Unsetenv("WEBHOOK_SECRET")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when TG_TOKEN is missing")
	}
	if err.Error() != "error: .env TG_TOKEN required" {
		t.Errorf("expected 'error: .env TG_TOKEN required', got '%v'", err)
	}
}

func TestLoadConfig_MissingClientID(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Unsetenv("TWITCH_CLIENT_ID")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("CALLBACK_URL")
		os.Unsetenv("WEBHOOK_SECRET")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when TWITCH_CLIENT_ID is missing")
	}
	if err.Error() != "error: .env TWITCH_CLIENT_ID required" {
		t.Errorf("expected 'error: .env TWITCH_CLIENT_ID required', got '%v'", err)
	}
}

func TestLoadConfig_MissingClientSecret(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Unsetenv("TWITCH_SECRET")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("CALLBACK_URL")
		os.Unsetenv("WEBHOOK_SECRET")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when TWITCH_SECRET is missing")
	}
	if err.Error() != "error: .env TWITCH_SECRET required" {
		t.Errorf("expected 'error: .env TWITCH_SECRET required', got '%v'", err)
	}
}

func TestLoadConfig_MissingCallbackUrl(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Unsetenv("CALLBACK_URL")
	os.Setenv("WEBHOOK_SECRET", "test_webhook_secret")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("WEBHOOK_SECRET")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when CALLBACK_URL is missing")
	}
	if err.Error() != "error: .env CALLBACK_URL required" {
		t.Errorf("expected 'error: .env CALLBACK_URL required', got '%v'", err)
	}
}

func TestLoadConfig_MissingWebhookSecret(t *testing.T) {
	os.Setenv("TG_TOKEN", "test_token")
	os.Setenv("TWITCH_CLIENT_ID", "test_client_id")
	os.Setenv("TWITCH_SECRET", "test_client_secret")
	os.Setenv("CALLBACK_URL", "https://example.com/webhook")
	os.Unsetenv("WEBHOOK_SECRET")
	defer func() {
		os.Unsetenv("TG_TOKEN")
		os.Unsetenv("TWITCH_CLIENT_ID")
		os.Unsetenv("TWITCH_SECRET")
		os.Unsetenv("CALLBACK_URL")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when WEBHOOK_SECRET is missing")
	}
	if err.Error() != "error: .env WEBHOOK_SECRET required" {
		t.Errorf("expected 'error: .env WEBHOOK_SECRET required', got '%v'", err)
	}
}
