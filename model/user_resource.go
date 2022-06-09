package model

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

func (ur *UserResource) UserOwnershipActive() bool {
	return len(ur.SourceNames) > 0 &&
		len(ur.ApplicationTypesNames) > 0 &&
		ur.userIDPresent() &&
		ur.userResourceOwnership()
}

func (ur *UserResource) userIDPresent() bool {
	return ur.User.UserID != "" && ur.User != nil
}

func (ur *UserResource) userResourceOwnership() bool {
	return ur.ResourceOwnership == "user"
}
