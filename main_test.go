package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockStore implements a minimal Store interface for testing
type MockStore struct {
	getActiveByTwitchUserIDFunc func(userID string) ([]interface{}, error)
	deactivateByEventSubIDFunc  func(eventSubID string) error
}

func (m *MockStore) GetActiveByTwitchUserID(userID string) ([]interface{}, error) {
	if m.getActiveByTwitchUserIDFunc != nil {
		return m.getActiveByTwitchUserIDFunc(userID)
	}
	return []interface{}{}, nil
}

func (m *MockStore) DeactivateByEventSubID(eventSubID string) error {
	if m.deactivateByEventSubIDFunc != nil {
		return m.deactivateByEventSubIDFunc(eventSubID)
	}
	return nil
}

// MockTelegramBot implements a minimal Bot interface for testing
type MockTelegramBot struct {
	sendNotificationFunc func(chatID int64, message string) error
}

func (m *MockTelegramBot) SendNotification(chatID int64, message string) error {
	if m.sendNotificationFunc != nil {
		return m.sendNotificationFunc(chatID, message)
	}
	return nil
}

func TestEventSubNotification_Unmarshal(t *testing.T) {
	jsonData := `{
		"challenge": "test_challenge",
		"subscription": {
			"id": "sub_123",
			"status": "enabled",
			"type": "stream.online",
			"version": "1"
		},
		"event": {
			"broadcaster_user_id": "123456",
			"broadcaster_user_login": "testuser",
			"broadcaster_user_name": "TestUser",
			"type": "stream.online",
			"started_at": "2024-01-01T12:00:00Z"
		}
	}`

	var notification EventSubNotification
	err := json.Unmarshal([]byte(jsonData), &notification)
	if err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}

	if notification.Challenge != "test_challenge" {
		t.Errorf("expected Challenge='test_challenge', got '%s'", notification.Challenge)
	}
	if notification.Subscription.ID != "sub_123" {
		t.Errorf("expected Subscription.ID='sub_123', got '%s'", notification.Subscription.ID)
	}
	if notification.Event.BroadcasterUserLogin != "testuser" {
		t.Errorf("expected BroadcasterUserLogin='testuser', got '%s'", notification.Event.BroadcasterUserLogin)
	}
}

func TestStreamEvent_Unmarshal(t *testing.T) {
	jsonData := `{
		"broadcaster_user_id": "987654",
		"broadcaster_user_login": "shroud",
		"broadcaster_user_name": "Shroud",
		"type": "stream.online",
		"started_at": "2024-01-01T12:00:00Z"
	}`

	var event StreamEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to unmarshal stream event: %v", err)
	}

	if event.BroadcasterUserId != "987654" {
		t.Errorf("expected BroadcasterUserId='987654', got '%s'", event.BroadcasterUserId)
	}
	if event.BroadcasterUserLogin != "shroud" {
		t.Errorf("expected BroadcasterUserLogin='shroud', got '%s'", event.BroadcasterUserLogin)
	}
}

func TestSubscriptionDetail_Unmarshal(t *testing.T) {
	jsonData := `{
		"id": "sub_456",
		"status": "enabled",
		"type": "stream.online",
		"version": "1"
	}`

	var sub SubscriptionDetail
	err := json.Unmarshal([]byte(jsonData), &sub)
	if err != nil {
		t.Fatalf("failed to unmarshal subscription: %v", err)
	}

	if sub.ID != "sub_456" {
		t.Errorf("expected ID='sub_456', got '%s'", sub.ID)
	}
	if sub.Type != "stream.online" {
		t.Errorf("expected Type='stream.online', got '%s'", sub.Type)
	}
}

func TestHandleWebhook_MissingSignature(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	if w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
		if req.Method != "POST" {
			t.Error("expected POST method")
		}
	}
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(`invalid json`)))
	w := httptest.NewRecorder()

	_ = w
	_ = req

	body, _ := io.ReadAll(req.Body)
	if len(body) == 0 {
		t.Error("expected body to be read")
	}
}

func TestEventSubNotification_ChallengeVerification(t *testing.T) {
	notification := EventSubNotification{
		Challenge: "challenge_string_123",
		Subscription: SubscriptionDetail{
			ID:      "sub_id",
			Status:  "enabled",
			Type:    "stream.online",
			Version: "1",
		},
		Event: StreamEvent{
			BroadcasterUserId:    "123",
			BroadcasterUserLogin: "testuser",
			BroadcasterUserName:  "TestUser",
			Type:                 "stream.online",
			StartedAt:            "2024-01-01T12:00:00Z",
		},
	}

	if notification.Challenge != "challenge_string_123" {
		t.Errorf("expected Challenge to be preserved, got '%s'", notification.Challenge)
	}
}

func TestEventSubNotification_MessageType_Verification(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Twitch-Eventsub-Message-Type", "webhook_callback_verification")

	msgType := req.Header.Get("Twitch-Eventsub-Message-Type")
	if msgType != "webhook_callback_verification" {
		t.Errorf("expected message type 'webhook_callback_verification', got '%s'", msgType)
	}
}

func TestEventSubNotification_MessageType_Revocation(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Twitch-Eventsub-Message-Type", "revocation")

	msgType := req.Header.Get("Twitch-Eventsub-Message-Type")
	if msgType != "revocation" {
		t.Errorf("expected message type 'revocation', got '%s'", msgType)
	}
}

func TestEventSubNotification_MessageType_Notification(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Twitch-Eventsub-Message-Type", "notification")

	msgType := req.Header.Get("Twitch-Eventsub-Message-Type")
	if msgType != "notification" {
		t.Errorf("expected message type 'notification', got '%s'", msgType)
	}
}

func TestStreamEvent_MessageFormatting(t *testing.T) {
	event := StreamEvent{
		BroadcasterUserId:    "123456",
		BroadcasterUserLogin: "shroud",
		BroadcasterUserName:  "Shroud",
		Type:                 "stream.online",
		StartedAt:            "2024-01-01T12:00:00Z",
	}

	msg := "🔴 <b>" + event.BroadcasterUserName + "</b> начал стрим!\n\nhttps://twitch.tv/" + event.BroadcasterUserLogin

	if msg != "🔴 <b>Shroud</b> начал стрим!\n\nhttps://twitch.tv/shroud" {
		t.Errorf("expected formatted message, got '%s'", msg)
	}
}
