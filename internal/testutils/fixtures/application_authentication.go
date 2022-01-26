package fixtures

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
)

var TestApplicationAuthentication = []m.ApplicationAuthentication{
	{
		ID:                1,
		CreatedAt:         CreatedAt,
		UpdatedAt:         UpdatedAt,
		VaultPath:         fmt.Sprintf("Application_%d_%v", TestTenantData[0].Id, TestAuthenticationData[0].ID),
		TenantID:          TestTenantData[0].Id,
		ApplicationID:     1,
		AuthenticationUID: TestAuthenticationData[0].ID,
	},
}
