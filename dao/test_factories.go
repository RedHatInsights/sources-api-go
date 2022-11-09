package dao

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func CreateTenantForAccountNumber(accountNumber string) (*int64, error) {
	identityStruct := identity.Identity{
		AccountNumber: accountNumber,
	}

	tenantDao := GetTenantDao()
	tenant, err := tenantDao.GetOrCreateTenant(&identityStruct)
	if err != nil {
		return nil, fmt.Errorf("error getting or creating the tenant")
	}

	return &tenant.Id, nil
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

type SourceOwnershipDataTestSuite struct {
	userA                       *m.User
	userB                       *m.User
	userWithoutOwnershipRecords *m.User
	resourcesUserA              *m.BulkCreateOutput
	resourcesUserB              *m.BulkCreateOutput
	resourcesNoUser             *m.BulkCreateOutput
}

func (s *SourceOwnershipDataTestSuite) TenantID() *int64 {
	return &s.userA.TenantID
}

func (s *SourceOwnershipDataTestSuite) GetRequestParamsUserA() *RequestParams {
	return &RequestParams{TenantID: s.TenantID(), UserID: &s.userA.Id}
}

func (s *SourceOwnershipDataTestSuite) GetRequestParamsUserB() *RequestParams {
	return &RequestParams{TenantID: s.TenantID(), UserID: &s.userB.Id}
}

func (s *SourceOwnershipDataTestSuite) UserA() *m.User {
	return s.userA
}

func (s *SourceOwnershipDataTestSuite) UserWithoutOwnership() *m.User {
	return s.userWithoutOwnershipRecords
}

func (s *SourceOwnershipDataTestSuite) SourceUserA() *m.Source {
	return &s.resourcesUserA.Sources[0]
}

func (s *SourceOwnershipDataTestSuite) SourceIDsUserA() []int64 {
	var sourcesIDs []int64
	for _, source := range s.resourcesUserA.Sources {
		sourcesIDs = append(sourcesIDs, source.ID)
	}

	for _, source := range s.resourcesNoUser.Sources {
		sourcesIDs = append(sourcesIDs, source.ID)
	}

	return sourcesIDs
}

func (s *SourceOwnershipDataTestSuite) SourceUserB() *m.Source {
	return &s.resourcesUserB.Sources[0]
}

func (s *SourceOwnershipDataTestSuite) SourceNoUser() *m.Source {
	return &s.resourcesNoUser.Sources[0]
}

func (s *SourceOwnershipDataTestSuite) SourceIDsNoUser() []int64 {
	var sourcesIDs []int64

	for _, source := range s.resourcesNoUser.Sources {
		sourcesIDs = append(sourcesIDs, source.ID)
	}

	return sourcesIDs
}

func (s *SourceOwnershipDataTestSuite) ApplicationUserA() *m.Application {
	return &s.resourcesUserA.Applications[0]
}

func (s *SourceOwnershipDataTestSuite) ApplicationNoUser() *m.Application {
	return &s.resourcesNoUser.Applications[0]
}

func (s *SourceOwnershipDataTestSuite) ApplicationUserB() *m.Application {
	return &s.resourcesUserB.Applications[0]
}

func (s *SourceOwnershipDataTestSuite) AuthenticationUserA() *m.Authentication {
	return &s.resourcesUserA.Authentications[0]
}

func (s *SourceOwnershipDataTestSuite) AuthenticationUserB() *m.Authentication {
	return &s.resourcesUserB.Authentications[0]
}

func (s *SourceOwnershipDataTestSuite) AuthenticationNoUser() *m.Authentication {
	return &s.resourcesNoUser.Authentications[0]
}

func (s *SourceOwnershipDataTestSuite) ApplicationAuthenticationUserA() *m.ApplicationAuthentication {
	return &s.resourcesUserA.ApplicationAuthentications[0]
}

func (s *SourceOwnershipDataTestSuite) ApplicationAuthenticationUserB() *m.ApplicationAuthentication {
	return &s.resourcesUserB.ApplicationAuthentications[0]
}

func (s *SourceOwnershipDataTestSuite) ApplicationAuthenticationNoUser() *m.ApplicationAuthentication {
	return &s.resourcesNoUser.ApplicationAuthentications[0]
}

func TestSuiteForSourceWithOwnership(performTest func(suiteData *SourceOwnershipDataTestSuite) error) error {
	accountNumber := "112567"
	userIDA := "userA"
	userIDB := "userB"
	userIDWithoutOwnershipRecords := "userWithoutRecords"

	applicationTypeID := fixtures.TestApplicationTypeData[3].Id
	sourceTypeID := fixtures.TestSourceTypeData[2].Id
	resourcesUserA, userA, err := CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, &userIDA)
	if err != nil {
		return fmt.Errorf("unable to create source with subresources: %v for user %v", err, userIDA)
	}

	testSuiteData := &SourceOwnershipDataTestSuite{}

	testSuiteData.resourcesUserA = resourcesUserA
	testSuiteData.userA = userA

	resourcesUserB, userB, err := CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, &userIDB)
	if err != nil {
		return fmt.Errorf("unable to create source with subresources: %v for user %v", err, userIDB)
	}

	testSuiteData.resourcesUserB = resourcesUserB
	testSuiteData.userB = userB

	recordsNoUser, _, err := CreateSourceWithSubResources(sourceTypeID, applicationTypeID, accountNumber, nil)
	if err != nil {
		return fmt.Errorf("unable to create source with subresources: %v", err)
	}

	testSuiteData.resourcesNoUser = recordsNoUser

	userWithoutOwnRecords, err := CreateUserForUserID(userIDWithoutOwnershipRecords, testSuiteData.userA.TenantID)
	if err != nil {
		return fmt.Errorf("unable to create user: %v", err)
	}

	testSuiteData.userWithoutOwnershipRecords = userWithoutOwnRecords

	err = performTest(testSuiteData)
	if err != nil {
		return err
	}

	return nil
}

func CreateSecretByName(name string, tenantID *int64, userID *int64) (*m.Authentication, error) {
	secretDao := GetSecretDao(&RequestParams{TenantID: tenantID})
	pass, _ := util.Encrypt("password")

	secret := &m.Authentication{
		Name:         util.StringRef(name),
		AuthType:     "X",
		Username:     util.StringRef("Y"),
		ResourceType: secretResourceType,
		ResourceID:   *tenantID,
		Password:     util.StringRef(pass),
	}

	if userID != nil {
		secret.UserID = userID
	}

	err := secretDao.Create(secret)
	return secret, err
}
