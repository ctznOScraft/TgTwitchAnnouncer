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
// It matches valid commands such as /start, /setchat, /subscribe, /stop, and /status.
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
/subscribe <twitch_login> — подписаться на уведомления о начале стрима
/stop <twitch_login> — отписаться от уведомлений по каналу
/status — список ваших текущих подписок
/setchat <chat_id> — изменить чат для всех подписок (по умолчанию уведомления приходят сюда)

Пример настройки:
1. /setchat -1001234567890 (опционально, если хотите отправлять в группу)
2. /subscribe shroud`
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

	err = h.store.UpdateUserChatID(msg.From.ID, chatID)
	if err != nil {
		return fmt.Sprintf("Ошибка обновления: %v", err)
	}

	return fmt.Sprintf("✅ Чат по умолчанию для ваших текущих подписок установлен: %d.\nНовые подписки всё равно необходимо будет создавать с учетом этого чата (или просто повторно вызовите /setchat).", chatID)
}

// handleSubscribe activates a subscription by checking if the Twitch channel exists,
// and subscribing via EventSub.
func (h *CommandHandler) handleSubscribe(msg *tgbotapi.Message) string {
	args := msg.CommandArguments()
	if args == "" {
		return "Укажите Twitch-канал. Пример: /subscribe shroud"
	}

	parts := strings.Fields(args)
	channelName := strings.ToLower(parts[0])

	userId, err := h.twitchClient.GetUserId(channelName)
	if err != nil {
		return fmt.Sprintf("Канал '%s' не найден на Twitch.", channelName)
	}

	sub, err := h.store.GetSubscription(msg.From.ID, channelName)
	if err != nil {
		if err != sql.ErrNoRows {
			return fmt.Sprintf("Ошибка базы данных: %v", err)
		}

		var chatID int64 = msg.From.ID
		allSubs, _ := h.store.GetAllByTelegramUser(msg.From.ID)
		if len(allSubs) > 0 {
			chatID = allSubs[0].TelegramChatID
		}

		sub = &storage.Subscription{
			TelegramUserID: msg.From.ID,
			TelegramChatID: chatID,
			TwitchChannel:  channelName,
			TwitchUserID:   userId,
			CreatedAt:      time.Now(),
		}
	}

	if sub.Active {
		return fmt.Sprintf("Вы уже подписаны на уведомления для канала %s.", sub.TwitchChannel)
	}

	activeSubs, err := h.store.GetActiveByTwitchUserID(sub.TwitchUserID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Sprintf("Ошибка базы данных: %v", err)
	}

	eventSubID := ""
	if len(activeSubs) > 0 && activeSubs[0].EventSubID != "" {
		eventSubID = activeSubs[0].EventSubID
	} else {
		eventSubID, err = h.twitchClient.SubscribeToStream(
			sub.TwitchUserID,
			h.callbackUrl,
			h.webhookSecret,
		)
		if err != nil {
			return fmt.Sprintf("Ошибка подписки на Twitch: %v", err)
		}
	}

	sub.EventSubID = eventSubID
	sub.Active = true

	err = h.store.UpsertSubscription(sub)
	if err != nil {
		return fmt.Sprintf("Ошибка сохранения: %v", err)
	}

	return fmt.Sprintf("✅ Подписка на %s активирована!\nЧат: %d", sub.TwitchChannel, sub.TelegramChatID)
}

// handleStop un-registers the subscription locally, and from Twitch EventSub if no one else needs it.
func (h *CommandHandler) handleStop(msg *tgbotapi.Message) string {
	args := msg.CommandArguments()
	if args == "" {
		return "Укажите Twitch-канал для отписки. Пример: /stop shroud"
	}

	channelName := strings.TrimSpace(strings.ToLower(args))

	sub, err := h.store.GetSubscription(msg.From.ID, channelName)
	if err != nil || !sub.Active {
		return fmt.Sprintf("Вы не подписаны на канал %s.", channelName)
	}

	err = h.store.Deactivate(msg.From.ID, channelName)
	if err != nil {
		return fmt.Sprintf("Ошибка деактивации: %v", err)
	}

	if sub.EventSubID != "" {
		activeSubs, _ := h.store.GetActiveByTwitchUserID(sub.TwitchUserID)
		if len(activeSubs) == 0 {
			err = h.twitchClient.DeleteSubscription(sub.EventSubID)
			if err != nil {
				log.Printf("Ошибка удаления EventSub %s: %v", sub.EventSubID, err)
			}
		}
	}

	return fmt.Sprintf("✅ Подписка на %s деактивирована.", channelName)
}

// handleStatus reports current settings and all active subscriptions to the user.
func (h *CommandHandler) handleStatus(msg *tgbotapi.Message) string {
	subs, err := h.store.GetAllByTelegramUser(msg.From.ID)
	if err != nil || len(subs) == 0 {
		return "У вас пока нет подписок. Начните с /subscribe <логин>."
	}

	var activeList []string
	var chatIDStr string

	for _, sub := range subs {
		if sub.Active {
			activeList = append(activeList, fmt.Sprintf("- %s", sub.TwitchChannel))
			chatIDStr = fmt.Sprintf("%d", sub.TelegramChatID)
		}
	}

	if len(activeList) == 0 {
		return "У вас нет активных подписок."
	}

	return fmt.Sprintf(
		"📋 Ваши активные подписки:\n%s\n\nTelegram чат для уведомлений: %s",
		strings.Join(activeList, "\n"), chatIDStr,
	)
}
