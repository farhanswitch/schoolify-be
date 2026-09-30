package utilities

import (
	"context"
	"time"

	"schoolify_be/connections"
)

type RedisUtility struct{}

var redisUtility RedisUtility

func (ru RedisUtility) SaveValue(key string, value interface{}, ttl time.Duration) error {
	var ctx context.Context = context.TODO()
	err := connections.ConnectRedis().Set(ctx, key, value, ttl).Err()
	return err
}
func (ru RedisUtility) DeleteValue(key string) error {
	var ctx context.Context = context.TODO()
	err := connections.ConnectRedis().Del(ctx, key).Err()
	return err
}
func (ru RedisUtility) GetValue(key string) (string, error) {
	var ctx context.Context = context.TODO()
	value, err := connections.ConnectRedis().Get(ctx, key).Result()
	return value, err
}
func RedisInstance() RedisUtility {
	if redisUtility == (RedisUtility{}) {
		redisUtility = RedisUtility{}
	}
	return redisUtility
}
