package mocks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/hashicorp/vault/api"
)

type MockVault struct {
}

var VaultPath = []string{fmt.Sprintf("Application_%d_%v", fixtures.TestTenantData[0].Id, fixtures.TestAuthenticationData[0].ID)}

func (m *MockVault) Read(path string) (*api.Secret, error) {
	if path != fmt.Sprintf("secret/data/%d/%v", fixtures.TestTenantData[0].Id, VaultPath[0]) {
		return nil, errors.New("boom")
	}

	secret := &api.Secret{}
	secret.Data = make(map[string]interface{})
	secret.Data["data"] = fixtures.TestAuthenticationVaultData
	secret.Data["metadata"] = fixtures.TestAuthenticationVaultMetaData

	return secret, nil
}

func (m *MockVault) List(_ string) (*api.Secret, error) {
	secret := &api.Secret{}

	secret.Data = make(map[string]interface{})
	secret.Data["keys"] = []interface{}{VaultPath[0]}

	return secret, nil
}

func (m *MockVault) Write(path string, data map[string]interface{}) (*api.Secret, error) {
	secret := &api.Secret{}
	secret.Data = data
	secret.Data["version"] = json.Number("2")

	return secret, nil
}

func (m *MockVault) Delete(path string) (*api.Secret, error) {
	if strings.HasSuffix(path, fixtures.TestAuthenticationData[0].ID) {
		secret := &api.Secret{}
		secret.Data = make(map[string]interface{})
		secret.Data["data"] = fixtures.TestAuthenticationVaultData
		secret.Data["metadata"] = fixtures.TestAuthenticationVaultMetaData

		return secret, nil
	}
	return nil, util.NewErrNotFound("authentication")
}
