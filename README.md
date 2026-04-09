# TgTwitchAnnouncer

Telegram бот для автоматических уведомлений о начале трансляций на Twitch.

## Возможности

- Автоматические уведомления когда стример начинает трансляцию
- Управление подписками через команды `/subscribe`, `/stop`
- Отправка уведомлений в личные чаты или группы
- Использование Twitch EventSub API для надежного получения событий
- SQLite база данных для хранения подписок
- Поддержка множественных пользователей и каналов

## Требования

- **Go** 1.23+
- **Twitch Developer Account** с зарегистрированным приложением
- **Telegram Bot Token** (создать через @BotFather)
- **Публичный HTTPS URL** для webhook'а

## Быстрый старт

### 1. Клонируйте репозиторий

```bash
git clone https://github.com/ctznOScraft/TgTwitchAnnouncer.git
cd TgTwitchAnnouncer
```

### 2. Установите зависимости

```bash
go mod download
```

### 3. Создайте .env файл

```bash
cp .env.example .env
```

Отредактируйте `.env` и заполните все необходимые параметры:

```dotenv
TG_TOKEN=your_token_here
TWITCH_CLIENT_ID=your_client_id_here
TWITCH_SECRET=your_secret_here
CALLBACK_URL=https://your-domain.com/webhook
WEBHOOK_SECRET=your_random_secret_here
PORT=4192
```

### 4. Запустите бота

```bash
go run main.go
```

Или скомпилируйте:

```bash
go build -o tgtwitchbot
./tgtwitchbot
```

## Команды Telegram бота

| Команда | Описание | Пример |
|---------|---------|--------|
| `/start` | Показать справку и список команд | `/start` |
| `/subscribe <логин>` | Подписаться на уведомления о стриме | `/subscribe shroud` |
| `/stop <логин>` | Отписаться от канала | `/stop shroud` |
| `/status` | Показать список активных подписок | `/status` |
| `/setchat <chat_id>` | Установить чат по умолчанию для уведомлений | `/setchat -1001234567890` |

## Архитектура проекта

```
TgTwitchAnnouncer/
├── main.go                    # Точка входа, HTTP server, webhook handler
├── internal/
│   ├── config/
│   │   └── config.go         # Загрузка конфигурации из .env
│   ├── storage/
│   │   └── storage.go        # SQLite операции с подписками
│   ├── telegram/
│   │   ├── telegram.go       # Telegram Bot API wrapper
│   │   └── handler.go        # Обработчик команд бота
│   └── twitch/
│       ├── twitch.go         # Twitch Helix API клиент
│       └── verify.go         # Верификация webhook подписей
├── .env                      # Конфигурация
├── .env.example              # Пример конфигурации
├── go.mod                    # Go модули
└── data.db                   # SQLite база
```

## База данных

Бот использует SQLite с одной таблицей `subscriptions`:

| Поле | Тип | Описание |
|------|-----|---------|
| `id` | INTEGER PRIMARY KEY | Уникальный ID подписки |
| `telegram_user_id` | INTEGER | ID пользователя в Telegram |
| `telegram_chat_id` | INTEGER | ID чата/канала для уведомлений |
| `twitch_channel` | TEXT | Логин канала (shroud, valorant, etc) |
| `twitch_user_id` | TEXT | Внутренний ID стримера на Twitch |
| `eventsub_id` | TEXT | ID подписки в Twitch EventSub |
| `active` | INTEGER | 1 - активна, 0 - деактивирована |
| `created_at` | DATETIME | Время создания подписки |
