package twitch

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"
)

func TestVerifySignature_Valid(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	message := []byte(msgID + timestamp)
	message = append(message, body...)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	expectedSig := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)
	headers.Set("Twitch-Eventsub-Message-Signature", expectedSig)

	if !VerifySignature(secret, headers, body) {
		t.Error("expected signature verification to pass")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)
	headers.Set("Twitch-Eventsub-Message-Signature", "sha256=invalidsignature")

	if VerifySignature(secret, headers, body) {
		t.Error("expected signature verification to fail")
	}
}

func TestVerifySignature_MissingMsgID(t *testing.T) {
	secret := "test_secret"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)
	headers.Set("Twitch-Eventsub-Message-Signature", "sha256=somesignature")

	if VerifySignature(secret, headers, body) {
		t.Error("expected signature verification to fail when Message-Id is missing")
	}
}

func TestVerifySignature_MissingTimestamp(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	body := []byte(`{"test":"data"}`)

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Signature", "sha256=somesignature")

	if VerifySignature(secret, headers, body) {
		t.Error("expected signature verification to fail when Timestamp is missing")
	}
}

func TestVerifySignature_MissingSignature(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)

	if VerifySignature(secret, headers, body) {
		t.Error("expected signature verification to fail when Signature is missing")
	}
}

func TestVerifySignature_DifferentSecret(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	message := []byte(msgID + timestamp)
	message = append(message, body...)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	originalSig := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)
	headers.Set("Twitch-Eventsub-Message-Signature", originalSig)

	if VerifySignature("different_secret", headers, body) {
		t.Error("expected signature verification to fail with different secret")
	}
}

func TestVerifySignature_ModifiedBody(t *testing.T) {
	secret := "test_secret"
	msgID := "12345"
	timestamp := "1234567890"
	body := []byte(`{"test":"data"}`)

	message := []byte(msgID + timestamp)
	message = append(message, body...)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	originalSig := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	headers := http.Header{}
	headers.Set("Twitch-Eventsub-Message-Id", msgID)
	headers.Set("Twitch-Eventsub-Message-Timestamp", timestamp)
	headers.Set("Twitch-Eventsub-Message-Signature", originalSig)

	modifiedBody := []byte(`{"test":"modified"}`)
	if VerifySignature(secret, headers, modifiedBody) {
		t.Error("expected signature verification to fail with modified body")
	}
}
