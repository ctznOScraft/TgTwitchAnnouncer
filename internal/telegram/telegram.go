package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot wraps the highly-used go-telegram-bot-api instance for convenience and abstraction.
type Bot struct {
	API *tgbotapi.BotAPI
}

// NewBot initializes a new Bot structure using the provided token.
// Returns an error if the connection to Telegram API fails.
func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &Bot{API: api}, nil
}

// SendNotification formats and sends a single text message to a given telegram chat ID.
// Uses HTML parsing mode.
func (b *Bot) SendNotification(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := b.API.Send(msg)
	return err
}

// GetUpdatesChan sets up and returns an updates channel to listen for incoming Telegram messages.
// Configures a timeout to establish long pooling.
func (b *Bot) GetUpdatesChan() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.API.GetUpdatesChan(u)
}
