package redis

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/go-redis/redis"
)

var Client *redis.Client

func Init() {
	cfg := config.Get()
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.CacheHost, cfg.CachePort),
		Password: cfg.CachePassword,
	})
}
