package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Subscription represents a single user's configuration for Twitch notifications.
type Subscription struct {
	ID             int64     // Primary key in the database.
	TelegramUserID int64     // The user ID of the person configuring the bot in Telegram.
	TelegramChatID int64     // The destination chat/channel ID where notifications will be sent (can be the user themselves).
	TwitchChannel  string    // The Twitch channel name (broadcaster login).
	TwitchUserID   string    // The internal Twitch User ID for the broadcaster.
	EventSubID     string    // The ID of the subscription registered with Twitch EventSub.
	Active         bool      // Flag indicating whether this subscription is currently active or disabled.
	CreatedAt      time.Time // Timestamp of when this subscription was originally created.
}

// Store encapsulates the database connection and operations for SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a new storage instance connected to the SQLite database path provided.
// Returns an error if the database connection cannot be established.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("storage: open db: %w", err)
	}
	return &Store{db: db}, nil
}

// Init sets up the necessary schema for the SQLite database, creating tables and indexes
// if they do not yet exist.
func (s *Store) Init() error {
	query := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_user_id INTEGER NOT NULL,
		telegram_chat_id INTEGER NOT NULL DEFAULT 0,
		twitch_channel   TEXT NOT NULL DEFAULT '',
		twitch_user_id   TEXT NOT NULL DEFAULT '',
		eventsub_id      TEXT NOT NULL DEFAULT '',
		active           INTEGER NOT NULL DEFAULT 0,
		created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_twitch_user_id ON subscriptions(twitch_user_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_telegram_user ON subscriptions(telegram_user_id);
	`
	_, err := s.db.Exec(query)
	return err
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertSubscription creates or updates a subscription for the given Telegram user.
// It relies on conflict resolution (ON CONFLICT) for telegram_user_id.
func (s *Store) UpsertSubscription(sub *Subscription) error {
	query := `
	INSERT INTO subscriptions (telegram_user_id, telegram_chat_id, twitch_channel, twitch_user_id, eventsub_id, active, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(telegram_user_id) DO UPDATE SET
		telegram_chat_id = excluded.telegram_chat_id,
		twitch_channel   = excluded.twitch_channel,
		twitch_user_id   = excluded.twitch_user_id,
		eventsub_id      = excluded.eventsub_id,
		active           = excluded.active
	`
	_, err := s.db.Exec(query,
		sub.TelegramUserID,
		sub.TelegramChatID,
		sub.TwitchChannel,
		sub.TwitchUserID,
		sub.EventSubID,
		sub.Active,
		sub.CreatedAt,
	)
	return err
}

// GetByTelegramUser retrieves the Subscription data associated with a specific Telegram User ID.
func (s *Store) GetByTelegramUser(telegramUserID int64) (*Subscription, error) {
	query := `SELECT id, telegram_user_id, telegram_chat_id, twitch_channel, twitch_user_id, eventsub_id, active, created_at
	           FROM subscriptions WHERE telegram_user_id = ?`
	row := s.db.QueryRow(query, telegramUserID)
	return scanSubscription(row)
}

// GetActiveByTwitchUserID retrieves all active subscriptions that are set up to notify
// for a specific Twitch User ID (broadcaster). Usually called when a stream.online event occurs.
func (s *Store) GetActiveByTwitchUserID(twitchUserID string) ([]Subscription, error) {
	query := `SELECT id, telegram_user_id, telegram_chat_id, twitch_channel, twitch_user_id, eventsub_id, active, created_at
	           FROM subscriptions WHERE twitch_user_id = ? AND active = 1`
	rows, err := s.db.Query(query, twitchUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubscriptions(rows)
}

// GetAllActive retrieves all currently active subscriptions across all users.
// Used primarily on application startup to ensure Twitch EventSub subscriptions are verified/re-created.
func (s *Store) GetAllActive() ([]Subscription, error) {
	query := `SELECT id, telegram_user_id, telegram_chat_id, twitch_channel, twitch_user_id, eventsub_id, active, created_at
	           FROM subscriptions WHERE active = 1`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubscriptions(rows)
}

// Deactivate disables a subscription for the given Telegram User ID by setting active = 0
// and unsetting its EventSubID, stopping further notifications.
func (s *Store) Deactivate(telegramUserID int64) error {
	_, err := s.db.Exec(`UPDATE subscriptions SET active = 0, eventsub_id = '' WHERE telegram_user_id = ?`, telegramUserID)
	return err
}

// scanSubscription is a helper that reads a single *sql.Row into a *Subscription struct.
func scanSubscription(row *sql.Row) (*Subscription, error) {
	var sub Subscription
	var active int
	err := row.Scan(
		&sub.ID,
		&sub.TelegramUserID,
		&sub.TelegramChatID,
		&sub.TwitchChannel,
		&sub.TwitchUserID,
		&sub.EventSubID,
		&active,
		&sub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	sub.Active = active == 1
	return &sub, nil
}

// scanSubscriptions is a helper that reads multiple rows into a slice of Subscription structs.
func scanSubscriptions(rows *sql.Rows) ([]Subscription, error) {
	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var active int
		err := rows.Scan(
			&sub.ID,
			&sub.TelegramUserID,
			&sub.TelegramChatID,
			&sub.TwitchChannel,
			&sub.TwitchUserID,
			&sub.EventSubID,
			&active,
			&sub.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		sub.Active = active == 1
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}
