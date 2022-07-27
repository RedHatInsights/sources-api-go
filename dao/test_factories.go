package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/google/uuid"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func CreateTenantForAccountNumber(accountNumber string) (*int64, error) {
	identityStruct := identity.Identity{
		AccountNumber: accountNumber,
	}

	tenantDao := GetTenantDao()
	tenantID, err := tenantDao.GetOrCreateTenantID(&identityStruct)
	if err != nil {
		return nil, fmt.Errorf("error getting or creating the tenant")
	}

	return &tenantID, nil
}

func CreateUserForUserID(userIDFromHeader string, tenantID int64) (*m.User, error) {
	userDao := GetUserDao(&tenantID)
	user, err := userDao.FindOrCreate(userIDFromHeader)
	if err != nil {
		return nil, fmt.Errorf("error getting or creating the user")
	}

	return user, nil
}

func CreateSource(sourceTypeID int64, tenantID int64, userID *int64) (*m.Source, error) {
	uid := uuid.New().String()
	name := uuid.New().String()
	source := &m.Source{Name: name, UserID: userID, SourceTypeID: sourceTypeID, Uid: &uid}
	requestParamsForCreate := &RequestParams{TenantID: &tenantID, UserID: userID}
	sourceDaoForCreate := GetSourceDao(requestParamsForCreate)
	err := sourceDaoForCreate.Create(source)
	if err != nil {
		return nil, fmt.Errorf("error creating the source")
	}

	return source, err
}

func CreateApplication(sourceID int64, applicationTypeID int64, tenantID int64, userID *int64) (*m.Application, error) {
	app := &m.Application{UserID: userID, ApplicationTypeID: applicationTypeID, SourceID: sourceID}
	requestParamsForCreate := &RequestParams{TenantID: &tenantID, UserID: userID}
	appDaoForCreate := GetApplicationDao(requestParamsForCreate)
	err := appDaoForCreate.Create(app)
	if err != nil {
		return nil, fmt.Errorf("error creating the application")
	}

	return app, err
}

func CreateAuthenticationFromApplication(appID int64, tenantID int64, userID *int64) (*m.Authentication, error) {
	name := uuid.New().String()
	auth := &m.Authentication{Name: &name, UserID: userID, ResourceType: "Application", ResourceID: appID}
	requestParamsForCreate := &RequestParams{TenantID: &tenantID, UserID: userID}
	authDaoForCreate := GetAuthenticationDao(requestParamsForCreate)
	err := authDaoForCreate.Create(auth)
	if err != nil {
		return nil, fmt.Errorf("error creating the application")
	}

	return auth, err
}

func CreateApplicationAuthentication(authID int64, appID int64, hello int64, userID *int64) (*m.ApplicationAuthentication, error) {
	aa := &m.ApplicationAuthentication{UserID: userID, ApplicationID: appID, AuthenticationID: authID}
	requestParamsForCreate := &RequestParams{TenantID: &hello, UserID: userID}
	appAuthDaoForCreate := GetApplicationAuthenticationDao(requestParamsForCreate)
	err := appAuthDaoForCreate.Create(aa)
	if err != nil {
		return nil, fmt.Errorf("error creating the application")
	}

	return aa, err
}

func CreateSourceWithSubResources(sourceTypeID int64, applicationTypeID int64, accountNumber string, userIDFromHeader *string) (*m.BulkCreateOutput, *m.User, error) {
	var bulkCreateOutput m.BulkCreateOutput

	tenantID, err := CreateTenantForAccountNumber(accountNumber)
	if err != nil {
		return nil, nil, err
	}

	var userID *int64
	var user *m.User

	if userIDFromHeader != nil {
		user, err = CreateUserForUserID(*userIDFromHeader, *tenantID)
		if err != nil {
			return nil, nil, err
		}
		userID = &user.Id
	}

	source, err := CreateSource(sourceTypeID, *tenantID, userID)
	if err != nil {
		return nil, nil, err
	}

	bulkCreateOutput.Sources = []m.Source{*source}

	app, err := CreateApplication(source.ID, applicationTypeID, *tenantID, userID)
	if err != nil {
		return nil, nil, err
	}

	bulkCreateOutput.Applications = []m.Application{*app}

	auth, err := CreateAuthenticationFromApplication(app.ID, *tenantID, userID)
	if err != nil {
		return nil, nil, err
	}

	bulkCreateOutput.Authentications = []m.Authentication{*auth}

	aa, err := CreateApplicationAuthentication(auth.DbID, app.ID, *tenantID, userID)
	if err != nil {
		return nil, nil, err
	}

	bulkCreateOutput.ApplicationAuthentications = []m.ApplicationAuthentication{*aa}

	return &bulkCreateOutput, user, nil
}
