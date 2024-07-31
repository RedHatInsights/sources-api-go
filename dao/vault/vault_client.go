package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

type VaultClient interface {
	Read(path string) (*api.Secret, error)
	List(path string) (*api.Secret, error)
	Write(path string, data map[string]interface{}) (*api.Secret, error)
	Delete(path string) (*api.Secret, error)
}

func NewClient() VaultClient {
	// Open up the conn to Vault
	cfg := api.DefaultConfig()
	if cfg == nil {
		panic("Failed to parse Vault Config")
	}
	err := cfg.ReadEnvironment()
	if err != nil {
		panic(fmt.Sprintf("Failed to read Vault Environment: %v", err))
	}

	vaultClient, err := api.NewClient(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to Create Vault Client: %v", err))
	}

	return vaultClient.Logical()
}
