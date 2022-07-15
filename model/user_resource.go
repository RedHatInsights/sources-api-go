package model

import "github.com/RedHatInsights/sources-api-go/util"

const UserOwnership = "user"

type UserResource struct {
	SourceNames           []string
	ApplicationTypesNames []string
	User                  *User
	ResourceOwnership     string
}

func (ur *UserResource) AddSourceAndApplicationTypeNames(sourceName, applicationTypeName string) {
	if ur.userResourceOwnership() {
		ur.SourceNames = append(ur.SourceNames, sourceName)
		ur.ApplicationTypesNames = append(ur.ApplicationTypesNames, applicationTypeName)
	}
}

func (ur *UserResource) userResourceOwnership() bool {
	return ur.ResourceOwnership == UserOwnership
}

func (ur *UserResource) UserOwnershipActive() bool {
	return len(ur.SourceNames) > 0 &&
		len(ur.ApplicationTypesNames) > 0 &&
		ur.userIDPresent() &&
		ur.userResourceOwnership()
}

func (ur *UserResource) userIDPresent() bool {
	return ur.User != nil && ur.User.UserID != ""
}

func (ur *UserResource) OwnershipPresentForSource(sourceName string) bool {
	if !ur.UserOwnershipActive() {
		return false
	}

	return util.SliceContainsString(ur.SourceNames, sourceName)
}
