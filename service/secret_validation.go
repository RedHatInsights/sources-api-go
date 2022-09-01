package service

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
)

func ValidateSecretCreationRequest(requestParams *dao.RequestParams, auth model.AuthenticationCreateRequest) error {
	if auth.Name != nil {
		if *auth.Name == "" {
			return fmt.Errorf("secret name have to be populated")
		}

		if dao.GetSecretDao(requestParams).NameExistsInCurrentTenant(*auth.Name) {
			return fmt.Errorf("secret name %s exists in current tenant", *auth.Name)
		}
	}

	if auth.ResourceIDRaw != nil {
		return fmt.Errorf("resource_id is not applicable to create secret")
	}

	if auth.ResourceType != "" && auth.ResourceType != dao.SecretResourceType {
		return fmt.Errorf("invalid resource_type - must be Tenant(default) or empty")
	}

	return nil
}
