package redis

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
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
	Client, err = valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", cfg.CacheHost, cfg.CachePort)},
		Password:    cfg.CachePassword,
		DisableCache: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize Valkey client: %v", err))
	}
}
