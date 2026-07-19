# flight-subscription

Сервис подписок на изменение цен авиабилетов. Пользователь подписывается на направление
(откуда/куда) с максимально приемлемой ценой. При поступлении события об изменении цены
система находит все активные подходящие подписки (то же направление и валюта, где
`max_price >= цены`) и создаёт для каждой уведомление.

## Архитектура

Два бинарника поверх общих слоёв (`internal/service`, `internal/storage/postgres`, `internal/http`, `internal/kafka*`):

- **`cmd/api`** — HTTP API. Принимает подписки и события об изменении цены, публикует
  события в Kafka, отдаёт уведомления. Graceful shutdown, автоматический прогон миграций.
- **`cmd/consumer`** — Kafka-воркер. Читает события `price.changed`, батчами подбирает
  подходящие подписки и создаёт уведомления. Retry с экспоненциальным backoff и
  dead-letter topic.

Инфраструктура: PostgreSQL (хранилище), Kafka (события), nginx (статика + прокси).

## HTTP API

| Метод | Путь | Назначение |
|-------|------|------------|
| GET  | `/healthz` | Health check (204) |
| POST | `/subscriptions` | Создать подписку |
| POST | `/price-events` | Опубликовать событие об изменении цены в Kafka (202) |
| GET  | `/notifications` | Уведомления по направлению (`from`, `to`, `limit`) |
| GET  | `/openapi.yaml` | OpenAPI-спецификация |
| GET  | `/swagger` | Swagger UI |

Схемы контрактов: REST — [`docs/openapi.yaml`](docs/openapi.yaml), события — [`docs/asyncapi.yaml`](docs/asyncapi.yaml).

## Запуск

Всё окружение через Docker Compose (Postgres, Kafka, инициализация топиков, API, web, 3 реплики consumer):

```sh
docker compose up --build
```

- API — http://localhost:8080 (Swagger: http://localhost:8080/swagger)
- Web UI — http://localhost:3000

Локально по отдельности (нужны доступные Postgres и Kafka):

```sh
go run ./cmd/api
go run ./cmd/consumer
```

## Конфигурация

Настраивается через переменные окружения (указаны значения по умолчанию):

| Переменная | По умолчанию |
|------------|--------------|
| `HTTP_ADDR` | `:8080` |
| `DATABASE_URL` | `postgres://app:app@localhost:5432/price_subscriptions?sslmode=disable` |
| `MIGRATIONS_DIR` | `internal/storage/postgres/migrations` |
| `DB_MAX_CONNS` | `25` |
| `KAFKA_BROKERS` | `localhost:9092` |
| `KAFKA_TOPIC` | `price.changed` |
| `KAFKA_GROUP_ID` | `price-subscriptions-consumer` |
| `KAFKA_DLQ_TOPIC` | `price.changed.dlq` |
| `KAFKA_MAX_RETRIES` | `5` |
| `KAFKA_INITIAL_BACKOFF` | `500ms` |
| `KAFKA_MAX_BACKOFF` | `30s` |
| `KAFKA_CONCURRENCY` | `8` |
| `NOTIFICATION_BATCH_SIZE` | `1000` |

## Тесты

```sh
go test ./...
```
