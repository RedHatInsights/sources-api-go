package mocks

import (
	"strings"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockApplicationDao struct {
	Applications []m.Application
}

func (mockAppDao *MockApplicationDao) SubCollectionList(primaryCollection interface{}, _, _ int, _ []util.Filter) ([]m.Application, int64, error) {
	var applications []m.Application

	switch object := primaryCollection.(type) {
	case m.Source:
		var sourceExists bool

		for _, s := range fixtures.TestSourceData {
			if s.ID == object.ID {
				sourceExists = true
			}
		}
		// if source doesn't exist, return Not Found Err
		if !sourceExists {
			return nil, 0, util.NewErrNotFound("source")
		}

		// else return list of related applications
		for _, app := range mockAppDao.Applications {
			if object.ID == app.SourceID {
				applications = append(applications, app)
			}
		}
	}

	return applications, int64(len(applications)), nil
}

func (mockAppDao *MockApplicationDao) List(_ int, _ int, _ []util.Filter) ([]m.Application, int64, error) {
	count := int64(len(mockAppDao.Applications))
	return mockAppDao.Applications, count, nil
}

func (mockAppDao *MockApplicationDao) GetById(id *int64) (*m.Application, error) {
	for _, app := range mockAppDao.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("application")
}

func (mockAppDao *MockApplicationDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Application, error) {
	for _, app := range mockAppDao.Applications {
		if app.ID == *id {
			for _, preload := range preloads {
				if strings.Contains(strings.ToLower(preload), "source") {
					app.Source = fixtures.TestSourceData[0]
				}
			}

			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("application")
}

func (mockAppDao *MockApplicationDao) Create(_ *m.Application) error {
	return nil
}

func (mockAppDao *MockApplicationDao) Update(_ *m.Application) error {
	return nil
}

func (mockAppDao *MockApplicationDao) Delete(id *int64) (*m.Application, error) {
	for _, app := range mockAppDao.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}
	return nil, util.NewErrNotFound("application")
}

func (mockAppDao *MockApplicationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (mockAppDao *MockApplicationDao) User() *int64 {
	user := int64(1)
	return &user
}

func (mockAppDao *MockApplicationDao) DeleteCascade(applicationId int64) ([]m.ApplicationAuthentication, *m.Application, error) {
	var application *m.Application
	for _, app := range fixtures.TestApplicationData {
		if app.ID == applicationId {
			application = &app
		}
	}

	if application == nil {
		return nil, nil, util.NewErrNotFound("application")
	}

	return fixtures.TestApplicationAuthenticationData, application, nil
}

func (mockAppDao *MockApplicationDao) Exists(applicationId int64) (bool, error) {
	for _, application := range mockAppDao.Applications {
		if application.ID == applicationId {
			return true, nil
		}
	}

	return false, nil
}

func (mockAppDao *MockApplicationDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (mockAppDao *MockApplicationDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (mockAppDao *MockApplicationDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (mockAppDao *MockApplicationDao) Pause(_ int64) error {
	return nil
}

func (mockAppDao *MockApplicationDao) Unpause(_ int64) error {
	return nil
}

func (mockAppDao *MockApplicationDao) IsSuperkey(_ int64) bool {
	return false
}
