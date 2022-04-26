package mocks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/hashicorp/vault/api"
)

type MockVault struct {
}

var VaultPath = []string{fmt.Sprintf("Application_%d_%v", fixtures.TestTenantData[0].Id, fixtures.TestAuthenticationData[0].ID)}

// getVaultPathsFromFixtures creates list of vault paths
func getVaultPathsFromFixtures() []interface{} {
	var vaultPaths []interface{}

	for _, auth := range fixtures.TestAuthenticationData {
		path := fmt.Sprintf("%s_%d_%s", auth.ResourceType, auth.TenantID, auth.ID)
		vaultPaths = append(vaultPaths, path)
	}
	return vaultPaths
}

// getIdFromVaultPath gets ID from given vault path
func getIdFromVaultPath(path string) string {
	paths := strings.Split(path, "_")
	uid := paths[len(paths)-1]

	return uid
}

// createSecretOutput creates and returns secret objects from given authentication
func createSecretOutput(auth m.Authentication) *api.Secret {
	secret := &api.Secret{}
	secret.Data = make(map[string]interface{})

	secret.Data["data"] = map[string]interface{}{
		"id":                  auth.ID,
		"tenant_id":           fmt.Sprintf("%d", auth.TenantID),
		"availability_status": *auth.AvailabilityStatus,
		"resource_type":       auth.ResourceType,
		"resource_id":         fmt.Sprintf("%d", auth.ResourceID),
		"source_id":           fmt.Sprintf("%d", auth.SourceID),
		"authtype":            "test",
		"name":                "OpenShift",
		"username":            "testusr",
		"extra":               map[string]interface{}{},
	}

	secret.Data["metadata"] = fixtures.TestAuthenticationVaultMetaData

	return secret
}

func (m *MockVault) Read(path string) (*api.Secret, error) {
	idFromPath := getIdFromVaultPath(path)

	for _, auth := range fixtures.TestAuthenticationData {
		if auth.ID == idFromPath {
			return createSecretOutput(auth), nil
		}
	}

	return nil, nil
}

func (m *MockVault) List(_ string) (*api.Secret, error) {
	secret := &api.Secret{}

	secret.Data = make(map[string]interface{})
	secret.Data["keys"] = getVaultPathsFromFixtures()

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
