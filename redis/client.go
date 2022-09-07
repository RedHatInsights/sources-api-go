package redis

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/go-redis/redis/v8"
)

var Client *redis.Client

// error used for checking if the error coming back from redis is nil or not
const NIL = redis.Nil

func Init() {
	cfg := config.Get()
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.CacheHost, cfg.CachePort),
		Password: cfg.CachePassword,
	})
}
