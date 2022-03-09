package service

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func ValidateApplicationAuthenticationCreateRequest(appAuth *m.ApplicationAuthenticationCreateRequest) error {
	appId, err := util.InterfaceToInt64(appAuth.ApplicationIDRaw)
	if err != nil {
		return err
	}
	appAuth.ApplicationID = appId

	authId, err := util.InterfaceToInt64(appAuth.AuthenticationIDRaw)
	if err != nil {
		return err
	}
	appAuth.AuthenticationID = authId

	return nil
}
