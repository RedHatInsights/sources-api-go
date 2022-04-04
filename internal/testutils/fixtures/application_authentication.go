package fixtures

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
)

var TestApplicationAuthenticationData = []m.ApplicationAuthentication{
	{
		ID:                1,
		VaultPath:         fmt.Sprintf("Application_%d_%v", TestTenantData[0].Id, TestAuthenticationData[0].ID),
		TenantID:          TestTenantData[0].Id,
		ApplicationID:     1,
		AuthenticationID:  1,
		AuthenticationUID: TestAuthenticationData[0].ID,
	},
	{
		ID:                2,
		VaultPath:         fmt.Sprintf("Application_%d_%v", TestTenantData[0].Id, TestAuthenticationData[0].ID),
		TenantID:          TestTenantData[0].Id,
		ApplicationID:     2,
		AuthenticationID:  1,
		AuthenticationUID: TestAuthenticationData[0].ID,
	},
}
