package connections

import (
	"context"
	"fmt"
	"log"
	"schoolify_be/configs"

	"github.com/redis/go-redis/v9"
)

var redisIntance *redis.Client

func ConnectRedis() *redis.Client {
	if redisIntance == nil {
		opt, err := redis.ParseURL(fmt.Sprintf("redis://%s:%d", configs.GetConfig().Redis.Host, configs.GetConfig().Redis.Port))
		if err != nil {
			log.Fatalf("Failed to create Redis Options.\nError: %s", err.Error())
		}
		client := redis.NewClient(opt)
		var ctx context.Context = context.TODO()
		err = client.Ping(ctx).Err()
		if err != nil {
			log.Fatalf("Failed to connect to Redis Server.\nError: %s", err.Error())
		}
		redisIntance = client
		log.Println("Redis connected!")
	}
	return redisIntance
}
