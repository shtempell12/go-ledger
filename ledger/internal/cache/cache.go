// Package cache — кеш значений в Redis с сериализацией в JSON.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const pingTimeout = 3 * time.Second

// Cache оборачивает клиент Redis.
type Cache struct {
	client *redis.Client
}

// Open подключается к Redis по REDIS_ADDR, REDIS_DB и REDIS_PASSWORD и проверяет соединение.
func Open() (*Cache, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	dbIndex := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_DB: %w", err)
		}
		dbIndex = parsed
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       dbIndex,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis %s: %w", addr, err)
	}
	return &Cache{client: client}, nil
}

// Get читает значение ключа в dst. Возвращает false, если ключа нет.
func (c *Cache) Get(ctx context.Context, key string, dst any) (bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get %s: %w", key, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return false, fmt.Errorf("decode cached %s: %w", key, err)
	}
	return true, nil
}

// Set записывает значение с TTL.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s for cache: %w", key, err)
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}
	return nil
}

// Del удаляет ключ.
func (c *Cache) Del(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del %s: %w", key, err)
	}
	return nil
}

// Close закрывает соединение.
func (c *Cache) Close() error {
	return c.client.Close()
}
