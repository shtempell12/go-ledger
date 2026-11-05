# Ledger

Бизнес-логика учёта расходов: транзакции, бюджеты, проверка лимитов и отчёт по категориям.
Данные хранятся в PostgreSQL, отчёты и список бюджетов кешируются в Redis на 30 секунд.
Ledger — Go-модуль, который импортирует Gateway. Бюджет считается по всем расходам категории,
а транзакции в категории без бюджета лимитом не ограничены.

## Создание базы и миграции

```bash
docker run -d --name cashcraft-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp -p 5432:5432 postgres:16
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine

go install github.com/pressly/goose/v3/cmd/goose@latest
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'

# Команды выполняются из корня репозитория
goose -dir ./ledger/migrations postgres "$DATABASE_URL" up
```

Откат:

```bash
goose -dir ./ledger/migrations postgres "$DATABASE_URL" down         # последняя миграция
goose -dir ./ledger/migrations postgres "$DATABASE_URL" down-to 0    # все миграции
```

Каждая миграция — один файл с секциями `-- +goose Up` и `-- +goose Down`: это формат Goose.
Применяйте миграции до первого запуска и после каждого изменения схемы.

## Переменные окружения

| Переменная | Значение по умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | — | строка подключения, имеет приоритет над `DB_*` |
| `DB_HOST`, `DB_PORT` | `localhost`, `5432` | адрес PostgreSQL |
| `DB_USER`, `DB_PASS` | `postgres`, `postgres` | учётные данные |
| `DB_NAME` | `cashapp` | имя базы, `sslmode=disable` |
| `REDIS_ADDR` | `localhost:6379` | адрес Redis |
| `REDIS_DB` | `0` | номер базы Redis |
| `REDIS_PASSWORD` | — | пароль Redis |

Если PostgreSQL недоступен, приложение завершается с понятной ошибкой. Если недоступен Redis,
оно пишет об этом в лог и работает напрямую с базой. Успешные подключения тоже логируются.

## Запуск

Основной сценарий — запуск через Gateway, см. [`gateway/README.md`](../gateway/README.md).
Отдельная команда проверяет подключения и печатает сохранённые данные:

```bash
cd ledger
go run ./cmd/ledger                # показать бюджеты и транзакции
go run ./cmd/ledger budgets.json   # сначала загрузить бюджеты из файла
```

## Тесты

Тесты, которым нужна база, запускаются только с `TEST_DATABASE_URL` и **очищают таблицы**,
поэтому используйте для них отдельную базу с применёнными миграциями. Без этой переменной
они пропускаются.

```bash
cd ledger
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable' go test -p 1 ./...
```
