package redis

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/lindgrenj6/sources-api-go/config"
)

var Client *redis.Client

func Init() {
	cfg := config.Get()
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.CacheHost, cfg.CachePort),
		Password: cfg.CachePassword,
	})
}
