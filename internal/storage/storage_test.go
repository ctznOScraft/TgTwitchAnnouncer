package storage

import (
	"os"
	"testing"
	"time"
)

// Helper function to create a test database
func setupTestDB(t *testing.T) *Store {
	dbFile := "test_data.db"
	os.Remove(dbFile)

	store, err := NewStore(dbFile)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	err = store.Init()
	if err != nil {
		t.Fatalf("failed to initialize test store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
		os.Remove(dbFile)
	})

	return store
}

func TestUpsertAndGetSubscription(t *testing.T) {
	store := setupTestDB(t)

	sub := &Subscription{
		TelegramUserID: 123,
		TelegramChatID: 456,
		TwitchChannel:  "testchannel",
		TwitchUserID:   "987",
		EventSubID:     "eventsub123",
		Active:         true,
		CreatedAt:      time.Now(),
	}

	err := store.UpsertSubscription(sub)
	if err != nil {
		t.Fatalf("failed to upsert subscription: %v", err)
	}

	retrieved, err := store.GetSubscription(123, "testchannel")
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}

	if retrieved.TelegramUserID != sub.TelegramUserID {
		t.Errorf("expected TelegramUserID=%d, got %d", sub.TelegramUserID, retrieved.TelegramUserID)
	}
	if retrieved.TwitchChannel != sub.TwitchChannel {
		t.Errorf("expected TwitchChannel=%s, got %s", sub.TwitchChannel, retrieved.TwitchChannel)
	}
	if retrieved.Active != sub.Active {
		t.Errorf("expected Active=%v, got %v", sub.Active, retrieved.Active)
	}
}

func TestGetAllByTelegramUser(t *testing.T) {
	store := setupTestDB(t)

	subs := []Subscription{
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel1",
			TwitchUserID:   "user1",
			EventSubID:     "event1",
			Active:         true,
			CreatedAt:      time.Now(),
		},
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel2",
			TwitchUserID:   "user2",
			EventSubID:     "event2",
			Active:         true,
			CreatedAt:      time.Now(),
		},
	}

	for _, sub := range subs {
		err := store.UpsertSubscription(&sub)
		if err != nil {
			t.Fatalf("failed to insert subscription: %v", err)
		}
	}

	retrieved, err := store.GetAllByTelegramUser(123)
	if err != nil {
		t.Fatalf("failed to get subscriptions: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(retrieved))
	}
}

func TestGetActiveByTwitchUserID(t *testing.T) {
	store := setupTestDB(t)

	activeSub := &Subscription{
		TelegramUserID: 123,
		TelegramChatID: 456,
		TwitchChannel:  "activeChannel",
		TwitchUserID:   "twitchUser1",
		EventSubID:     "event1",
		Active:         true,
		CreatedAt:      time.Now(),
	}

	inactiveSub := &Subscription{
		TelegramUserID: 124,
		TelegramChatID: 457,
		TwitchChannel:  "inactiveChannel",
		TwitchUserID:   "twitchUser1",
		EventSubID:     "event2",
		Active:         false,
		CreatedAt:      time.Now(),
	}

	store.UpsertSubscription(activeSub)
	store.UpsertSubscription(inactiveSub)

	retrieved, err := store.GetActiveByTwitchUserID("twitchUser1")
	if err != nil {
		t.Fatalf("failed to get active subscriptions: %v", err)
	}

	if len(retrieved) != 1 {
		t.Errorf("expected 1 active subscription, got %d", len(retrieved))
	}

	if !retrieved[0].Active {
		t.Error("expected subscription to be active")
	}
}

func TestGetAllActive(t *testing.T) {
	store := setupTestDB(t)

	subs := []Subscription{
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel1",
			TwitchUserID:   "user1",
			EventSubID:     "event1",
			Active:         true,
			CreatedAt:      time.Now(),
		},
		{
			TelegramUserID: 124,
			TelegramChatID: 457,
			TwitchChannel:  "channel2",
			TwitchUserID:   "user2",
			EventSubID:     "event2",
			Active:         false,
			CreatedAt:      time.Now(),
		},
		{
			TelegramUserID: 125,
			TelegramChatID: 458,
			TwitchChannel:  "channel3",
			TwitchUserID:   "user3",
			EventSubID:     "event3",
			Active:         true,
			CreatedAt:      time.Now(),
		},
	}

	for _, sub := range subs {
		store.UpsertSubscription(&sub)
	}

	retrieved, err := store.GetAllActive()
	if err != nil {
		t.Fatalf("failed to get all active subscriptions: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("expected 2 active subscriptions, got %d", len(retrieved))
	}

	for _, sub := range retrieved {
		if !sub.Active {
			t.Error("expected all subscriptions to be active")
		}
	}
}

func TestDeactivate(t *testing.T) {
	store := setupTestDB(t)

	sub := &Subscription{
		TelegramUserID: 123,
		TelegramChatID: 456,
		TwitchChannel:  "testchannel",
		TwitchUserID:   "987",
		EventSubID:     "eventsub123",
		Active:         true,
		CreatedAt:      time.Now(),
	}

	store.UpsertSubscription(sub)

	err := store.Deactivate(123, "testchannel")
	if err != nil {
		t.Fatalf("failed to deactivate subscription: %v", err)
	}

	retrieved, err := store.GetSubscription(123, "testchannel")
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}

	if retrieved.Active {
		t.Error("expected subscription to be inactive")
	}
	if retrieved.EventSubID != "" {
		t.Error("expected EventSubID to be cleared")
	}
}

func TestDeactivateByEventSubID(t *testing.T) {
	store := setupTestDB(t)

	subs := []Subscription{
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel1",
			TwitchUserID:   "user1",
			EventSubID:     "sameEventSubID",
			Active:         true,
			CreatedAt:      time.Now(),
		},
		{
			TelegramUserID: 124,
			TelegramChatID: 457,
			TwitchChannel:  "channel2",
			TwitchUserID:   "user2",
			EventSubID:     "sameEventSubID",
			Active:         true,
			CreatedAt:      time.Now(),
		},
	}

	for _, sub := range subs {
		store.UpsertSubscription(&sub)
	}

	err := store.DeactivateByEventSubID("sameEventSubID")
	if err != nil {
		t.Fatalf("failed to deactivate by EventSubID: %v", err)
	}

	retrieved, err := store.GetAllActive()
	if err != nil {
		t.Fatalf("failed to get active subscriptions: %v", err)
	}

	if len(retrieved) != 0 {
		t.Errorf("expected 0 active subscriptions, got %d", len(retrieved))
	}
}

func TestUpdateUserChatID(t *testing.T) {
	store := setupTestDB(t)

	subs := []Subscription{
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel1",
			TwitchUserID:   "user1",
			EventSubID:     "event1",
			Active:         true,
			CreatedAt:      time.Now(),
		},
		{
			TelegramUserID: 123,
			TelegramChatID: 456,
			TwitchChannel:  "channel2",
			TwitchUserID:   "user2",
			EventSubID:     "event2",
			Active:         true,
			CreatedAt:      time.Now(),
		},
	}

	for _, sub := range subs {
		store.UpsertSubscription(&sub)
	}

	err := store.UpdateUserChatID(123, 789)
	if err != nil {
		t.Fatalf("failed to update chat ID: %v", err)
	}

	retrieved, err := store.GetAllByTelegramUser(123)
	if err != nil {
		t.Fatalf("failed to get subscriptions: %v", err)
	}

	for _, sub := range retrieved {
		if sub.TelegramChatID != 789 {
			t.Errorf("expected TelegramChatID=789, got %d", sub.TelegramChatID)
		}
	}
}

func TestUpsertSubscription_Update(t *testing.T) {
	store := setupTestDB(t)

	sub := &Subscription{
		TelegramUserID: 123,
		TelegramChatID: 456,
		TwitchChannel:  "testchannel",
		TwitchUserID:   "987",
		EventSubID:     "event1",
		Active:         true,
		CreatedAt:      time.Now(),
	}

	store.UpsertSubscription(sub)

	sub.TelegramChatID = 999
	sub.EventSubID = "event2"
	sub.Active = false

	err := store.UpsertSubscription(sub)
	if err != nil {
		t.Fatalf("failed to upsert subscription: %v", err)
	}

	retrieved, err := store.GetSubscription(123, "testchannel")
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}

	if retrieved.TelegramChatID != 999 {
		t.Errorf("expected TelegramChatID=999, got %d", retrieved.TelegramChatID)
	}
	if retrieved.EventSubID != "event2" {
		t.Errorf("expected EventSubID='event2', got '%s'", retrieved.EventSubID)
	}
	if retrieved.Active {
		t.Error("expected subscription to be inactive")
	}
}
