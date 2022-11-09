package model

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/config"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
)

var ErrBadSecretStore = fmt.Errorf("invalid secret-store: %s", config.Get().SecretStore)

// fetches the secret-store dependent ID from the authentication
func (auth *Authentication) GetID() string {
	switch config.Get().SecretStore {
	case config.DatabaseStore, config.SecretsManagerStore:
		return strconv.FormatInt(auth.DbID, 10)

	case config.VaultStore:
		return auth.ID

	default:
		panic(ErrBadSecretStore)
	}
}

// fetches the secret-store dependent extra field from the authentication
func (auth *Authentication) GetExtra() map[string]interface{} {
	switch config.Get().SecretStore {
	case config.DatabaseStore, config.SecretsManagerStore:
		var extra map[string]interface{}

		if auth.ExtraDb != nil {
			err := json.Unmarshal(auth.ExtraDb, &extra)
			if err != nil {
				l.Log.Warnf("failed to unmarshal extra: %v", err)
			}
		}

		return extra

	case config.VaultStore:
		return auth.Extra

	default:
		panic(ErrBadSecretStore)
	}
}

// self-explanatory, the password can be in a different format based on the
// secret-store.
func (auth *Authentication) GetPassword() (*string, error) {
	switch config.Get().SecretStore {
	case config.DatabaseStore:
		if auth.Password == nil {
			return nil, nil
		}

		decrypted, err := util.Decrypt(*auth.Password)
		return &decrypted, err

	case config.VaultStore:
		return auth.Password, nil

	case config.SecretsManagerStore:
		return auth.Password, nil

	default:
		return nil, ErrBadSecretStore
	}
}

// set the extra field on the authentication struct based on which secret store we're using
func (auth *Authentication) SetExtra(extra map[string]interface{}) error {
	if extra == nil {
		return nil
	}

	switch config.Get().SecretStore {
	case config.DatabaseStore, config.SecretsManagerStore:
		var err error
		auth.ExtraDb, err = json.Marshal(extra)
		if err != nil {
			return err
		}

	case config.VaultStore:
		auth.Extra = extra

	default:
		return ErrBadSecretStore
	}

	return nil
}

// sets a field on the extra column, changing how to get to the field based on
// the secret store.
func (auth *Authentication) SetExtraField(key string, value interface{}) error {
	switch config.Get().SecretStore {
	case config.DatabaseStore, config.SecretsManagerStore:
		var err error
		var extra = make(map[string]interface{})

		if auth.ExtraDb != nil && string(auth.ExtraDb) != `null` {
			err := json.Unmarshal(auth.ExtraDb, &extra)
			if err != nil {
				return err
			}
		}

		extra[key] = value
		auth.ExtraDb, err = json.Marshal(extra)
		return err

	case config.VaultStore:
		if auth.Extra == nil {
			auth.Extra = make(map[string]interface{})
		}

		auth.Extra[key] = value

		return nil

	default:
		return ErrBadSecretStore
	}
}

func (auth *Authentication) SetPassword(pass *string) error {
	// if there isn't a password - just return since we don't have anything to do.
	if pass == nil {
		return nil
	}

	switch config.Get().SecretStore {
	case config.DatabaseStore:
		encrypted, err := util.Encrypt(*pass)
		if err != nil {
			return err
		}

		auth.Password = &encrypted
		return nil

	case config.VaultStore:
		auth.Password = pass
		return nil

	case config.SecretsManagerStore:
		auth.Password = pass
		return nil

	default:
		return ErrBadSecretStore
	}
}
