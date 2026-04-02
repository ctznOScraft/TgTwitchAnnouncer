package main

import (
	"TgTwitchAnnouncer/internal/config"
	"TgTwitchAnnouncer/internal/storage"
	"TgTwitchAnnouncer/internal/telegram"
	"TgTwitchAnnouncer/internal/twitch"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SubscriptionDetail contains details about the Twitch EventSub subscription.
type SubscriptionDetail struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// StreamEvent represents the payload of a stream.online event from Twitch.
type StreamEvent struct {
	BroadcasterUserId    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
	BroadcasterUserName  string `json:"broadcaster_user_name"`
	Type                 string `json:"type"`
	StartedAt            string `json:"started_at"`
}

// EventSubNotification represents the root structure of the Twitch EventSub webhook notification payload.
type EventSubNotification struct {
	Challenge    string             `json:"challenge"`
	Subscription SubscriptionDetail `json:"subscription"`
	Event        StreamEvent        `json:"event"`
}

// handleWebhook processes incoming HTTP requests from Twitch EventSub webhooks.
// It verifies the signature, handles challenge verification for subscription creation,
// and processes stream.online events by sending Telegram notifications.
func handleWebhook(
	w http.ResponseWriter,
	r *http.Request,
	tgBot *telegram.Bot,
	store *storage.Store,
	webhookSecret string,
) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Ошибка чтения тела запроса: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if !twitch.VerifySignature(webhookSecret, r.Header, body) {
		log.Println("Неверная подпись webhook запроса")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var notification EventSubNotification
	err = json.Unmarshal(body, &notification)
	if err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	messageType := r.Header.Get("Twitch-Eventsub-Message-Type")

	if messageType == "webhook_callback_verification" {
		w.Header().Set("Content-Type", "text/plain")
		_, err = w.Write([]byte(notification.Challenge))
		if err != nil {
			log.Printf("Ошибка отправки challenge: %v", err)
		}
		log.Println("Challenge получен и отправлен обратно")
		return
	}

	if messageType == "revocation" {
		log.Printf("Подписка Twitch отозвана! ID: %s, Причина: %s", notification.Subscription.ID,
			notification.Subscription.Status)
		err = store.DeactivateByEventSubID(notification.Subscription.ID)
		if err != nil {
			log.Printf("Ошибка деактивации отозванной подписки: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if messageType == "notification" && notification.Subscription.Type == "stream.online" {
		broadcasterID := notification.Event.BroadcasterUserId
		broadcasterLogin := notification.Event.BroadcasterUserLogin
		broadcasterName := notification.Event.BroadcasterUserName

		log.Printf("Стрим начался: %s (%s)", broadcasterName, broadcasterLogin)

		subs, err := store.GetActiveByTwitchUserID(broadcasterID)
		if err != nil {
			log.Printf("Ошибка получения подписок: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		msg := fmt.Sprintf("🔴 <b>%s</b> начал стрим!\n\nhttps://twitch.tv/%s",
			broadcasterName, broadcasterLogin)

		for _, sub := range subs {
			err = tgBot.SendNotification(sub.TelegramChatID, msg)
			if err != nil {
				log.Printf("Ошибка отправки в чат %d: %v", sub.TelegramChatID, err)
			} else {
				log.Printf("Уведомление отправлено в чат %d", sub.TelegramChatID)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// resubscribeActive checks all active subscriptions and re-creates EventSub subscriptions if needed.
// This function is useful for restoring subscriptions upon application startup.
func resubscribeActive(store *storage.Store, twitchClient *twitch.Client, cfg *config.Config) {
	time.Sleep(2 * time.Second)

	subs, err := store.GetAllActive()
	if err != nil {
		log.Printf("Ошибка загрузки активных подписок: %v", err)
		return
	}

	seenStreamers := make(map[string]string)

	for _, sub := range subs {
		if existingEventSubID, ok := seenStreamers[sub.TwitchUserID]; ok {
			sub.EventSubID = existingEventSubID
			_ = store.UpsertSubscription(&sub)
			continue
		}

		log.Printf("Переподписка на канал %s (user_id: %s)", sub.TwitchChannel, sub.TwitchUserID)

		if sub.EventSubID != "" {
			_ = twitchClient.DeleteSubscription(sub.EventSubID)
		}

		eventSubID, err := twitchClient.SubscribeToStream(
			sub.TwitchUserID,
			cfg.CallbackUrl,
			cfg.WebhookSecret,
		)
		if err != nil {
			log.Printf("Ошибка переподписки на %s: %v", sub.TwitchChannel, err)
			continue
		}

		seenStreamers[sub.TwitchUserID] = eventSubID
		sub.EventSubID = eventSubID
		err = store.UpsertSubscription(&sub)
		if err != nil {
			log.Printf("Ошибка обновления подписки в БД: %v", err)
		}
	}

	if len(subs) > 0 {
		log.Printf("Переподписка завершена: %d подписок", len(subs))
	} else {
		log.Println("Нет активных подписок для переподписки")
	}
}

// main is the entry point of the application. It initializes configuration,
// database connection, Twitch and Telegram clients, sets up active subscriptions,
// and starts the HTTP server to receive payloads from Twitch.
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	store, err := storage.NewStore("data.db")
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer store.Close()

	err = store.Init()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	twitchClient := twitch.NewClient(cfg.TwitchClientID, cfg.TwitchClientSecret)
	err = twitchClient.Auth()
	if err != nil {
		log.Fatalf("Ошибка авторизации Twitch: %v", err)
	}
	log.Println("Twitch авторизация успешна")

	tgBot, err := telegram.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Ошибка создания Telegram бота: %v", err)
	}
	log.Printf("Telegram бот авторизован: %s", tgBot.API.Self.UserName)

	go resubscribeActive(store, twitchClient, cfg)

	cmdHandler := telegram.NewCommandHandler(
		tgBot, store, twitchClient,
		cfg.CallbackUrl, cfg.WebhookSecret,
	)
	go cmdHandler.Start()
	log.Println("Обработчик Telegram-команд запущен")

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(w, r, tgBot, store, cfg.WebhookSecret)
	})

	addr := ":" + cfg.Port
	log.Printf("Сервер запущен на порту %s", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
