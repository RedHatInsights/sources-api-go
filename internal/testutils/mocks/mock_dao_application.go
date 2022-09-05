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

func (a *MockApplicationDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Application, int64, error) {
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
		for _, app := range a.Applications {
			if object.ID == app.SourceID {
				applications = append(applications, app)
			}
		}
	}

	return applications, int64(len(applications)), nil
}

func (a *MockApplicationDao) List(limit int, offset int, filters []util.Filter) ([]m.Application, int64, error) {
	count := int64(len(a.Applications))
	return a.Applications, count, nil
}

func (a *MockApplicationDao) GetById(id *int64) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("application")
}

func (a *MockApplicationDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Application, error) {
	for _, app := range a.Applications {
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

func (a *MockApplicationDao) Create(src *m.Application) error {
	return nil
}

func (a *MockApplicationDao) Update(src *m.Application) error {
	return nil
}

func (a *MockApplicationDao) Delete(id *int64) (*m.Application, error) {
	for _, app := range a.Applications {
		if app.ID == *id {
			return &app, nil
		}
	}
	return nil, util.NewErrNotFound("application")
}

func (a *MockApplicationDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (a *MockApplicationDao) User() *int64 {
	user := int64(1)
	return &user
}

func (a *MockApplicationDao) DeleteCascade(applicationId int64) ([]m.ApplicationAuthentication, *m.Application, error) {
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

func (a *MockApplicationDao) Exists(applicationId int64) (bool, error) {
	for _, application := range a.Applications {
		if application.ID == applicationId {
			return true, nil
		}
	}

	return false, nil
}

func (m *MockApplicationDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockApplicationDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockApplicationDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (a *MockApplicationDao) Pause(_ int64) error {
	return nil
}

func (a *MockApplicationDao) Unpause(_ int64) error {
	return nil
}

func (src *MockApplicationDao) IsSuperkey(id int64) bool {
	return false
}
