package mocks

import (
	"fmt"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/hashicorp/vault/api"
)

type MockLogical struct {
}

func (m MockLogical) Read(path string) (*api.Secret, error) {
	if path != fmt.Sprintf("secret/data/%d/%v", fixtures.TestTenantData[0].Id, "") {
		return nil, nil
	}

	secret := &api.Secret{}
	secret.Data = make(map[string]interface{})
	secret.Data["data"] = map[string]interface{}{}
	secret.Data["metadata"] = map[string]interface{}{}

	return secret, nil
}

func (m MockLogical) List(_ string) (*api.Secret, error) {
	secret := &api.Secret{}

	secret.Data = make(map[string]interface{})
	secret.Data["keys"] = []interface{}{""}

	return secret, nil
}

func (m MockLogical) Write(path string, data map[string]interface{}) (*api.Secret, error) {
	panic("implement me Write")
}

func (m MockLogical) Delete(path string) (*api.Secret, error) {
	panic("implement me Delete")
}
