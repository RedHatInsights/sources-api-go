package redis

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/valkey-io/valkey-go"
)

var Client valkey.Client

// IsNil checks if the error is a Valkey nil error (key does not exist)
func IsNil(err error) bool {
	return valkey.IsValkeyNil(err)
}

func Init() {
	cfg := config.Get()

	var err error

	address := fmt.Sprintf("%s:%d", cfg.CacheHost, cfg.CachePort)
	l.Log.Infof("Connecting to Redis/Valkey at %s...", address)

	Client, err = valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{address},
		Password:     cfg.CachePassword,
		DisableCache: true,
	})
	if err != nil {
		l.Log.Errorf("Failed to connect to Redis/Valkey at %s: %v", address, err)
		panic(fmt.Sprintf("failed to initialize Valkey client: %v", err))
	}

	l.Log.Infof("Redis/Valkey connection established at %s", address)
}
