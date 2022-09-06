package service

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
)

func ValidateSecretCreationRequest(requestParams *dao.RequestParams, auth model.SecretCreateRequest) error {
	if auth.Name != nil {
		if *auth.Name == "" {
			return fmt.Errorf("secret name have to be populated")
		}

		if dao.GetSecretDao(requestParams).NameExistsInCurrentTenant(*auth.Name) {
			return fmt.Errorf("secret name %s exists in current tenant", *auth.Name)
		}
	}
	return nil
}
