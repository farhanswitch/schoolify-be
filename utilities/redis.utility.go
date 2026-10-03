package utilities

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"schoolify_be/connections"
)

type RedisUtility struct {
	client *redis.Client
}

var (
	redisUtil     *RedisUtility
	redisUtilOnce sync.Once
)

// RedisInstance returns the singleton RedisUtility instance.
func RedisInstance() *RedisUtility {
	redisUtilOnce.Do(func() {
		redisUtil = &RedisUtility{
			client: connections.ConnectRedis(),
		}
	})
	return redisUtil
}

// SaveValue saves a key-value pair in Redis with a specified Time-to-Live (TTL).
func (r *RedisUtility) SaveValue(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// DeleteValue deletes a key from Redis.
func (r *RedisUtility) DeleteValue(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// GetValue retrieves a value by key from Redis.
func (r *RedisUtility) GetValue(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// GetTTL retrieves the remaining Time-To-Live (TTL) of a key from Redis.
func (r *RedisUtility) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}
