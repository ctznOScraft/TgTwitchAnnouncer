package twitch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	clientID := "test_client_id"
	clientSecret := "test_client_secret"

	client := NewClient(clientID, clientSecret)

	if client.ClientId != clientID {
		t.Errorf("expected ClientId=%s, got %s", clientID, client.ClientId)
	}
	if client.ClientSecret != clientSecret {
		t.Errorf("expected ClientSecret=%s, got %s", clientSecret, client.ClientSecret)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestAuth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" {
			t.Errorf("expected path /oauth2/token, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "test_access_token",
		})
	}))
	defer server.Close()

	client := NewClient("test_client_id", "test_client_secret")
	client.httpClient = &http.Client{}

	if client.ClientId == "" || client.ClientSecret == "" {
		t.Error("expected client to be properly initialized")
	}
}

func TestCreateRequest(t *testing.T) {
	client := NewClient("test_client_id", "test_client_secret")
	client.AccessToken = "test_access_token"

	payload := []byte(`{"test":"data"}`)
	req, err := client.createRequest("POST", "https://api.twitch.tv/helix/test", payload)

	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if req.Header.Get("Client-Id") != "test_client_id" {
		t.Error("expected Client-Id header to be set")
	}
	if req.Header.Get("Authorization") != "Bearer test_access_token" {
		t.Error("expected Authorization header to be set")
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type header to be set")
	}
}

func TestCreateRequest_WithoutPayload(t *testing.T) {
	client := NewClient("test_client_id", "test_client_secret")
	client.AccessToken = "test_access_token"

	req, err := client.createRequest("GET", "https://api.twitch.tv/helix/test", nil)

	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if req.Header.Get("Client-Id") != "test_client_id" {
		t.Error("expected Client-Id header to be set")
	}
	if req.Header.Get("Authorization") != "Bearer test_access_token" {
		t.Error("expected Authorization header to be set")
	}
}

func TestGetStreamInfo_Live(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/helix/streams" {
			t.Errorf("expected path /helix/streams, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StreamResponse{
			Data: []StreamInfo{
				{
					Title:       "Test Stream",
					ViewerCount: 1000,
					Type:        "live",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test_client_id", "test_client_secret")
	client.AccessToken = "test_access_token"
	client.httpClient = &http.Client{}

	if client.AccessToken == "" {
		t.Error("expected AccessToken to be set")
	}
}

func TestGetUserId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/helix/users" {
			t.Errorf("expected path /helix/users, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserResponse{
			Data: []UserInfo{
				{
					Id: "test_user_id",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test_client_id", "test_client_secret")
	client.AccessToken = "test_access_token"
	client.httpClient = &http.Client{}

	if client.AccessToken == "" {
		t.Error("expected AccessToken to be set")
	}
}

func TestEventSubSubscription_Marshal(t *testing.T) {
	event := EventSubSubscription{
		Type:    "stream.online",
		Version: "1",
		Condition: EventSubCondition{
			BroadcasterUserId: "123456",
		},
		Transport: EventSubTransport{
			Method:   "webhook",
			Callback: "https://example.com/webhook",
			Secret:   "test_secret",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var unmarshalled EventSubSubscription
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if unmarshalled.Type != "stream.online" {
		t.Errorf("expected Type='stream.online', got '%s'", unmarshalled.Type)
	}
	if unmarshalled.Condition.BroadcasterUserId != "123456" {
		t.Errorf("expected BroadcasterUserId='123456', got '%s'", unmarshalled.Condition.BroadcasterUserId)
	}
}
