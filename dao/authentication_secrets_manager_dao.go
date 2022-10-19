package dao

import (
	m "github.com/RedHatInsights/sources-api-go/model"
)

type authenticationSecretsManagerDaoImpl struct {
	// tenant ID + User ID + context information
	*RequestParams

	// embedded isntance of the db - since we use it for all fields except the
	// password. the only glaring difference is that we will store an
	// unencrypted ARN as the password field, so we can just fetch it directly
	// from amazon when we need to decrypt it.
	//
	// we also will continue to use all list/info methods as passthrough - only
	// overriding when necessary.
	authenticationDaoDbImpl
}

func (a *authenticationSecretsManagerDaoImpl) Create(src *m.Authentication) error {
	panic("not implemented") // TODO: Implement
}

func (a *authenticationSecretsManagerDaoImpl) BulkCreate(src *m.Authentication) error {
	panic("not implemented") // TODO: Implement
}

func (a *authenticationSecretsManagerDaoImpl) Update(src *m.Authentication) error {
	panic("not implemented") // TODO: Implement
}

func (a *authenticationSecretsManagerDaoImpl) Delete(id string) (*m.Authentication, error) {
	panic("not implemented") // TODO: Implement
}

// BulkDelete deletes all the authentications given as a list, and returns the ones that were deleted.
func (a *authenticationSecretsManagerDaoImpl) BulkDelete(authentications []m.Authentication) ([]m.Authentication, error) {
	panic("not implemented") // TODO: Implement
}
