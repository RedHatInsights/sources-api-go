package mocks

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/hashicorp/vault/api"
)

type MockVault struct {
}

var VaultPath = []string{fmt.Sprintf("Application_%d_%v", fixtures.TestTenantData[0].Id, fixtures.TestAuthenticationData[0].ID)}

func (m *MockVault) Read(path string) (*api.Secret, error) {
	if path != fmt.Sprintf("secret/data/%d/%v", fixtures.TestTenantData[0].Id, VaultPath[0]) {
		return nil, nil
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
	panic("implement me Write")
}

func (m *MockVault) Delete(path string) (*api.Secret, error) {
	panic("implement me Delete")
}
