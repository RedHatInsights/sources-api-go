package dao

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
)

type AuthenticationDaoImpl struct {
	TenantID *int64
}

func (a *AuthenticationDaoImpl) List(limit int, offset int, filters []middleware.Filter) ([]m.Authentication, int64, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, 0, err
	}

	end := 0
	if limit > len(keys) {
		end = len(keys)
	} else {
		end = limit
	}

	out := make([]m.Authentication, 0, len(keys))
	for _, val := range keys[offset:end] {
		secret, err := a.getKey(fmt.Sprintf("secret/data/%d/%s", *a.TenantID, val))
		if err != nil {
			return nil, 0, err
		}

		out = append(out, *secret)
	}
	count := int64(len(out))

	return out, count, nil
}

func (a *AuthenticationDaoImpl) GetById(uid string) (*m.Authentication, error) {
	keys, err := a.listKeys()
	if err != nil {
		return nil, err
	}
	var fullKey string
	for _, key := range keys {
		if strings.HasSuffix(key, uid) {
			fullKey = key
			break
		}
	}

	if fullKey == "" {
		return nil, fmt.Errorf("authentication not found")
	}

	return a.getKey(fmt.Sprintf("secret/data/%d/%s", *a.TenantID, fullKey))
}

func (a *AuthenticationDaoImpl) Create(auth *m.Authentication) error {
	auth.ID = uuid.New().String()
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	_, err = Vault.Write(path, data)
	if err != nil {
		return err
	}

	return nil
}

func (a *AuthenticationDaoImpl) Update(auth *m.Authentication) error {
	path := fmt.Sprintf("secret/data/%d/%s_%v_%s", *a.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)

	data, err := auth.ToVaultMap()
	if err != nil {
		return err
	}

	_, err = Vault.Write(path, data)
	if err != nil {
		return err
	}

	return nil
}

func (a *AuthenticationDaoImpl) Delete(uid string) error {
	keys, err := a.listKeys()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if strings.HasSuffix(key, uid) {
			path := fmt.Sprintf("secret/metadata/%d/%s", *a.TenantID, key)
			out, err := Vault.Delete(path)
			fmt.Println(out)
			return err
		}
	}

	return fmt.Errorf("not found")
}

func (a *AuthenticationDaoImpl) Tenant() *int64 {
	return a.TenantID
}

func (a *AuthenticationDaoImpl) listKeys() ([]string, error) {
	path := fmt.Sprintf("secret/metadata/%d/", *a.TenantID)
	list, err := Vault.List(path)
	if err != nil || list == nil {
		return nil, err
	}

	var data []interface{}
	var ok bool
	if data, ok = list.Data["keys"].([]interface{}); !ok {
		return nil, fmt.Errorf("bad data came back from vault")
	}

	keys := make([]string, len(data))
	for i, key := range data {
		if keys[i], ok = key.(string); !ok {
			return nil, fmt.Errorf("bad type cast")
		}
	}

	return keys, nil
}

func (a *AuthenticationDaoImpl) getKey(path string) (*m.Authentication, error) {
	paths := strings.Split(path, "_")
	uid := paths[len(paths)-1]

	secret, err := Vault.Read(path)
	if err != nil || secret == nil {
		return nil, fmt.Errorf("authentication not found")
	}

	auth := authFromVault(secret)
	if auth == nil {
		return nil, fmt.Errorf("failed to deserialize secret from vault")
	}

	auth.ID = uid
	return auth, nil
}

func authFromVault(secret *api.Secret) *m.Authentication {
	var data, metadata, extra map[string]interface{}
	var ok bool
	if data, ok = secret.Data["data"].(map[string]interface{}); !ok {
		return nil
	}
	if metadata, ok = secret.Data["metadata"].(map[string]interface{}); !ok {
		return nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, metadata["created_time"].(string))
	if err != nil {
		return nil
	}

	if data["extra"] != nil {
		if extra, ok = data["extra"].(map[string]interface{}); !ok {
			return nil
		}
	}

	auth := &m.Authentication{}
	auth.CreatedAt = createdAt
	auth.Version = metadata["version"].(json.Number).String()

	if extra != nil {
		auth.Extra = extra
	}

	if data["name"] != nil {
		if auth.Name, ok = data["name"].(string); !ok {
			return nil
		}
	}
	if data["authtype"] != nil {
		if auth.AuthType, ok = data["authtype"].(string); !ok {
			return nil
		}
	}
	if data["username"] != nil {
		if auth.Username, ok = data["username"].(string); !ok {
			return nil
		}
	}
	if data["password"] != nil {
		if auth.Password, ok = data["password"].(string); !ok {
			return nil
		}
	}
	if data["resource_type"] != nil {
		if auth.ResourceType, ok = data["resource_type"].(string); !ok {
			return nil
		}
	}
	if data["resource_id"] != nil {
		id, err := strconv.ParseInt(data["resource_id"].(string), 10, 64)
		if err != nil {
			return nil
		}
		auth.ResourceID = id
	}

	return auth
}
