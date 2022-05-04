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
		AuthenticationID:  TestAuthenticationData[0].DbID,
		AuthenticationUID: TestAuthenticationData[0].ID,
	},
	{
		ID:                2,
		VaultPath:         fmt.Sprintf("Application_%d_%v", TestTenantData[0].Id, TestAuthenticationData[3].ID),
		TenantID:          TestTenantData[0].Id,
		ApplicationID:     5,
		AuthenticationID:  TestAuthenticationData[3].DbID,
		AuthenticationUID: TestAuthenticationData[3].ID,
	},
	{
		ID:                300,
		VaultPath:         fmt.Sprintf("Application_%d_%v", TestTenantData[0].Id, TestAuthenticationData[4].ID),
		TenantID:          TestTenantData[0].Id,
		ApplicationID:     4,
		AuthenticationID:  TestAuthenticationData[4].DbID,
		AuthenticationUID: TestAuthenticationData[4].ID,
	},
}
