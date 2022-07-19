package model

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
