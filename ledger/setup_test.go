package ledger

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

// Тесты очищают таблицы, поэтому база задаётся только явно через TEST_DATABASE_URL.
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		log.Println("TEST_DATABASE_URL is not set, tests that need a database are skipped")
		os.Exit(m.Run())
	}
	os.Setenv("DATABASE_URL", dsn)
	if err := Init(); err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	Close()
	os.Exit(code)
}

// reset очищает таблицы и кеш перед тестом.
func reset(t *testing.T) {
	t.Helper()
	if conn == nil {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if _, err := conn.Exec(`TRUNCATE budgets, expenses RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if store != nil {
		if err := store.Del(context.Background(), budgetsCacheKey); err != nil {
			t.Fatalf("clear cache: %v", err)
		}
	}
}

func clearReportCache(t *testing.T, from, to time.Time) {
	t.Helper()
	if err := store.Del(context.Background(), reportCacheKey(from, to)); err != nil {
		t.Fatalf("clear cache: %v", err)
	}
}
