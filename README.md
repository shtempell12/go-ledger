# CashCraft: Gateway и Ledger

Учебный проект по курсу «Язык Go».

- `gateway/` — HTTP API на порту `:8080`. Импортирует `ledger` как Go-модуль (`replace` в `gateway/go.mod`).
- `ledger/` — бизнес-логика: транзакции, бюджеты, проверка лимитов.

## Запуск

Нужен Go 1.22 или новее.

```bash
cd gateway
go run .
```

Тесты запускаются в каждом модуле отдельно: `cd ledger && go test ./...`, `cd gateway && go test ./...`.

## Эндпоинты

| Метод и путь | Описание | Ответ |
|---|---|---|
| `POST /api/transactions` | добавить транзакцию | `201`, `400` (валидация), `409` (превышен бюджет) |
| `GET /api/transactions` | список транзакций, сначала новые | `200` |
| `POST /api/budgets` | создать или обновить бюджет категории | `201`, `400` |
| `GET /api/budgets` | список бюджетов | `200` |

Все ответы идут с `Content-Type: application/json; charset=utf-8`. Ошибки имеют формат
`{"error":"описание"}`, внутренние ошибки возвращают `500`.

## Примеры cURL

```bash
curl -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -d '{"category":"еда","limit":5000}'

curl -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'

# Превышение бюджета: ответ 409 {"error":"budget exceeded"}
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":10000,"category":"еда","description":"банкет","date":"2025-09-11"}'

curl http://localhost:8080/api/transactions
curl http://localhost:8080/api/budgets
```
