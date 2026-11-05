# CashCraft: Gateway и Ledger

Учебный проект по курсу «Язык Go».

- [`gateway/`](gateway/README.md) — HTTP API на порту `:8080`. Импортирует `ledger` как Go-модуль
  (`replace` в `gateway/go.mod`).
- [`ledger/`](ledger/README.md) — бизнес-логика, хранение в PostgreSQL, кеш отчётов в Redis.

Gateway обращается к тем же HTTP-эндпоинтам, что и раньше, но Ledger теперь хранит данные
в PostgreSQL, а отчёты кешируются в Redis на короткий TTL.

## Быстрый старт

Нужны Go 1.26+, Docker и Goose.

```bash
docker run -d --name cashcraft-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine

go install github.com/pressly/goose/v3/cmd/goose@latest
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
goose -dir ./ledger/migrations postgres "$DATABASE_URL" up

cd gateway
go run .
```

Эндпоинты и примеры cURL описаны в [`gateway/README.md`](gateway/README.md),
переменные окружения и откат миграций — в [`ledger/README.md`](ledger/README.md).

```bash
curl -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'
```
