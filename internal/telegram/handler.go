package telegram

import (
	"TgTwitchAnnouncer/internal/storage"
	"TgTwitchAnnouncer/internal/twitch"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler is responsible for processing incoming Telegram commands
// and orchestrating actions between the Telegram bot, local storage, and the Twitch API.
type CommandHandler struct {
	bot           *Bot
	store         *storage.Store
	twitchClient  *twitch.Client
	callbackUrl   string
	webhookSecret string
}

// NewCommandHandler creates and returns a new CommandHandler instance.
func NewCommandHandler(
	bot *Bot,
	store *storage.Store,
	twitchClient *twitch.Client,
	callbackUrl string,
	webhookSecret string,
) *CommandHandler {
	return &CommandHandler{
		bot:           bot,
		store:         store,
		twitchClient:  twitchClient,
		callbackUrl:   callbackUrl,
		webhookSecret: webhookSecret,
	}
}

// Start begins listening to the Telegram event stream and blocks to process incoming messages.
// It matches valid commands such as /start, /setchannel, /setchat, /subscribe, /stop, and /status.
func (h *CommandHandler) Start() {
	updates := h.bot.GetUpdatesChan()
	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		msg := update.Message
		if msg.Chat.Type != "private" {
			continue
		}

		var reply string
		switch msg.Command() {
		case "start":
			reply = h.handleStart()
		case "setchannel":
			reply = h.handleSetChannel(msg)
		case "setchat":
			reply = h.handleSetChat(msg)
		case "subscribe":
			reply = h.handleSubscribe(msg)
		case "stop":
			reply = h.handleStop(msg)
		case "status":
			reply = h.handleStatus(msg)
		default:
			reply = "Неизвестная команда. Используйте /start для списка команд."
		}

		resp := tgbotapi.NewMessage(msg.Chat.ID, reply)
		_, err := h.bot.API.Send(resp)
		if err != nil {
			log.Printf("Ошибка отправки ответа: %v", err)
		}
	}
}

// handleStart returns the welcoming help text showing usage examples.
func (h *CommandHandler) handleStart() string {
	return `Привет! Я бот для уведомлений о стримах на Twitch.

Команды:
/setchannel <twitch_login> — указать Twitch-канал
/setchat <chat_id> — указать Telegram чат/канал для уведомлений
/subscribe — подписаться на уведомления
/stop — отписаться от уведомлений
/status — текущие настройки

Пример настройки:
1. /setchannel shroud
2. /setchat -1001234567890
3. /subscribe`
}

// handleSetChannel parses the given twitch login and stores it for the user if it exists.
func (h *CommandHandler) handleSetChannel(msg *tgbotapi.Message) string {
	args := msg.CommandArguments()
	if args == "" {
		return "Укажите логин Twitch-канала. Пример: /setchannel shroud"
	}

	channelName := strings.TrimSpace(strings.ToLower(args))

	userId, err := h.twitchClient.GetUserId(channelName)
	if err != nil {
		return fmt.Sprintf("Канал '%s' не найден на Twitch: %v", channelName, err)
	}

	sub, err := h.store.GetByTelegramUser(msg.From.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			return fmt.Sprintf("Ошибка базы данных: %v", err)
		}
		sub = &storage.Subscription{
			TelegramUserID: msg.From.ID,
			CreatedAt:      time.Now(),
		}
	}

	sub.TwitchChannel = channelName
	sub.TwitchUserID = userId

	err = h.store.UpsertSubscription(sub)
	if err != nil {
		return fmt.Sprintf("Ошибка сохранения: %v", err)
	}

	return fmt.Sprintf("✅ Twitch-канал установлен: %s (ID: %s)", channelName, userId)
}

// handleSetChat configures the destination chat or channel ID for the user's stream notifications.
func (h *CommandHandler) handleSetChat(msg *tgbotapi.Message) string {
	args := msg.CommandArguments()
	if args == "" {
		return "Укажите ID чата/канала. Пример: /setchat -1001234567890\n\nЧтобы узнать ID, добавьте бота в чат/канал и перешлите мне оттуда любое сообщение, или используйте @userinfobot."
	}

	chatID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		return "Неверный формат ID чата. ID должен быть числом."
	}

	sub, err := h.store.GetByTelegramUser(msg.From.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			return fmt.Sprintf("Ошибка базы данных: %v", err)
		}
		sub = &storage.Subscription{
			TelegramUserID: msg.From.ID,
			CreatedAt:      time.Now(),
		}
	}

	sub.TelegramChatID = chatID

	err = h.store.UpsertSubscription(sub)
	if err != nil {
		return fmt.Sprintf("Ошибка сохранения: %v", err)
	}

	return fmt.Sprintf("✅ Чат для уведомлений установлен: %d", chatID)
}

// handleSubscribe activates the configured subscription by dispatching a subscribe
// request to the Twitch EventSub system and marking it as active locally.
func (h *CommandHandler) handleSubscribe(msg *tgbotapi.Message) string {
	sub, err := h.store.GetByTelegramUser(msg.From.ID)
	if err != nil {
		return "Сначала настройте канал и чат с помощью /setchannel и /setchat"
	}

	if sub.TwitchChannel == "" {
		return "Сначала укажите Twitch-канал: /setchannel <login>"
	}
	if sub.TelegramChatID == 0 {
		return "Сначала укажите чат для уведомлений: /setchat <chat_id>"
	}

	if sub.Active {
		return "Вы уже подписаны на уведомления. Используйте /stop чтобы отписаться."
	}

	eventSubID, err := h.twitchClient.SubscribeToStream(
		sub.TwitchUserID,
		h.callbackUrl,
		h.webhookSecret,
	)
	if err != nil {
		return fmt.Sprintf("Ошибка подписки на Twitch: %v", err)
	}

	sub.EventSubID = eventSubID
	sub.Active = true

	err = h.store.UpsertSubscription(sub)
	if err != nil {
		return fmt.Sprintf("Ошибка сохранения: %v", err)
	}

	return fmt.Sprintf("✅ Подписка активирована!\nКанал: %s\nЧат: %d\nEventSub ID: %s",
		sub.TwitchChannel, sub.TelegramChatID, eventSubID)
}

// handleStop un-registers the subscription from Twitch EventSub and marks it deactivated locally.
func (h *CommandHandler) handleStop(msg *tgbotapi.Message) string {
	sub, err := h.store.GetByTelegramUser(msg.From.ID)
	if err != nil {
		return "У вас нет активных подписок."
	}

	if !sub.Active {
		return "У вас нет активных подписок."
	}

	if sub.EventSubID != "" {
		err = h.twitchClient.DeleteSubscription(sub.EventSubID)
		if err != nil {
			log.Printf("Ошибка удаления EventSub %s: %v", sub.EventSubID, err)
		}
	}

	err = h.store.Deactivate(msg.From.ID)
	if err != nil {
		return fmt.Sprintf("Ошибка деактивации: %v", err)
	}

	return "✅ Подписка деактивирована."
}

// handleStatus reports current settings to the user.
func (h *CommandHandler) handleStatus(msg *tgbotapi.Message) string {
	sub, err := h.store.GetByTelegramUser(msg.From.ID)
	if err != nil {
		return "У вас пока нет настроек. Начните с /setchannel и /setchat."
	}

	status := "❌ Неактивна"
	if sub.Active {
		status = "✅ Активна"
	}

	chatStr := "не задан"
	if sub.TelegramChatID != 0 {
		chatStr = fmt.Sprintf("%d", sub.TelegramChatID)
	}

	channelStr := "не задан"
	if sub.TwitchChannel != "" {
		channelStr = sub.TwitchChannel
	}

	return fmt.Sprintf(
		"📋 Ваши настройки:\n\nTwitch-канал: %s\nTelegram чат: %s\nПодписка: %s",
		channelStr, chatStr, status,
	)
}
