// Package db открывает подключение к PostgreSQL.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // драйвер "pgx" для database/sql
)

const pingTimeout = 5 * time.Second

// Open подключается к базе по настройкам из окружения и проверяет соединение.
func Open() (*sql.DB, error) {
	conn, err := sql.Open("pgx", DSN())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return conn, nil
}

// DSN возвращает DATABASE_URL, а если он не задан, собирает строку из DB_HOST, DB_PORT,
// DB_USER, DB_PASS и DB_NAME со значениями по умолчанию для локальной разработки.
func DSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(env("DB_USER", "postgres"), env("DB_PASS", "postgres")),
		Host:     net.JoinHostPort(env("DB_HOST", "localhost"), env("DB_PORT", "5432")),
		Path:     env("DB_NAME", "cashapp"),
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
