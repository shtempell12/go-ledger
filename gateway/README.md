# Gateway

HTTP API на порту `:8080`. Эндпоинты остались прежними, но Ledger теперь хранит данные
в PostgreSQL, поэтому они переживают перезапуск. Отчёты кешируются в Redis на 30 секунд.
Ledger импортируется как Go-модуль и работает в одном процессе с Gateway, поэтому запуск
Gateway запускает и Ledger, а перезапуск Gateway перезапускает оба.

## Запуск

Сначала создайте базу и примените миграции: см. [`ledger/README.md`](../ledger/README.md).

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
export REDIS_ADDR=localhost:6379 REDIS_DB=0 REDIS_PASSWORD=

cd gateway
go run .
```

## Эндпоинты

| Метод и путь | Описание | Ответ |
|---|---|---|
| `POST /api/transactions` | добавить транзакцию | `201`, `400` (валидация), `409` (превышен бюджет) |
| `GET /api/transactions` | список транзакций, сначала новые | `200` |
| `POST /api/budgets` | создать или обновить бюджет категории | `201`, `400` |
| `GET /api/budgets` | список бюджетов | `200` |
| `GET /api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | расходы по категориям за период | `200`, `400` |

Все ответы идут с `Content-Type: application/json; charset=utf-8`. Ошибки имеют формат
`{"error":"описание"}`, внутренние ошибки возвращают `500`.

## Примеры cURL

```bash
curl -i -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -d '{"category":"еда","limit":5000}'

curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'

# Превышение бюджета: ответ 409 {"error":"budget exceeded"}, запись не добавляется
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":10000,"category":"еда","description":"банкет","date":"2025-09-11"}'

curl http://localhost:8080/api/transactions
curl http://localhost:8080/api/budgets
```

Эффект кеша: выполните запрос дважды подряд. В логе первый вызов даст `report cache miss`,
второй — `report cache hit`.

```bash
curl 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
curl 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
```

## Проверка сохранности данных

Остановите Gateway, запустите его снова и повторите `GET /api/transactions` и `GET /api/budgets`:
записи останутся на месте.
