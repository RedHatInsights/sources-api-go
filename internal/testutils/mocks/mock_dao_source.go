package mocks

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockSourceDao struct {
	Sources        []m.Source
	RelatedSources []m.Source
}

func (mockSourceDao *MockSourceDao) SubCollectionList(primaryCollection interface{}, _, _ int, _ []util.Filter) ([]m.Source, int64, error) {
	var sources []m.Source

	switch object := primaryCollection.(type) {
	case m.SourceType:
		var sourceTypeExists bool

		for _, sourceType := range fixtures.TestSourceTypeData {
			if sourceType.Id == object.Id {
				sourceTypeExists = true
			}
		}

		// Source type doesn't exist = return Not Found err
		if !sourceTypeExists {
			return nil, 0, util.NewErrNotFound("source type")
		}

		// Source type exists = return sources subcollection
		for index, source := range mockSourceDao.Sources {
			if source.SourceTypeID == object.Id {
				sources = append(sources, mockSourceDao.Sources[index])
			}
		}

	case m.ApplicationType:
		var appTypeExists bool

		for _, appType := range fixtures.TestApplicationTypeData {
			if appType.Id == object.Id {
				appTypeExists = true
			}
		}

		// Application type doesn't exist = return Not Found err
		if !appTypeExists {
			return nil, 0, util.NewErrNotFound("application type")
		}

		// Application type exists = find related sources
		sources = testutils.GetSourcesWithAppType(object.Id)

	default:
		return nil, 0, fmt.Errorf("unexpected primary collection type")
	}

	count := int64(len(sources))

	return sources, count, nil
}

func (mockSourceDao *MockSourceDao) List(_, _ int, _ []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(mockSourceDao.Sources))
	return mockSourceDao.Sources, count, nil
}

func (mockSourceDao *MockSourceDao) ListInternal(_, _ int, _ []util.Filter, _ bool) ([]m.Source, int64, error) {
	count := int64(len(mockSourceDao.Sources))
	return mockSourceDao.Sources, count, nil
}

func (mockSourceDao *MockSourceDao) GetById(id *int64) (*m.Source, error) {
	for _, i := range mockSourceDao.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (mockSourceDao *MockSourceDao) Create(s *m.Source) error {
	mockSourceDao.Sources = append(mockSourceDao.Sources, *s)
	return nil
}

func (mockSourceDao *MockSourceDao) Update(_ *m.Source) error {
	return nil
}

func (mockSourceDao *MockSourceDao) Delete(id *int64) (*m.Source, error) {
	for i, source := range mockSourceDao.Sources {
		if source.ID == *id {
			mockSourceDao.Sources = append(mockSourceDao.Sources[:i], mockSourceDao.Sources[i+1:]...)

			return &source, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (mockSourceDao *MockSourceDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (mockSourceDao *MockSourceDao) User() *int64 {
	user := int64(1)
	return &user
}

func (mockSourceDao *MockSourceDao) Exists(sourceId int64) (bool, error) {
	for _, source := range mockSourceDao.Sources {
		if source.ID == sourceId {
			return true, nil
		}
	}

	return false, nil
}

// NameExistsInCurrentTenant returns always false because it's the safe default in case the request gets validated
// in the tests.
func (mockSourceDao *MockSourceDao) NameExistsInCurrentTenant(_ string) bool {
	return false
}

func (mockSourceDao *MockSourceDao) IsSuperkey(_ int64) bool {
	return false
}

func (mockSourceDao *MockSourceDao) GetByIdWithPreload(id *int64, _ ...string) (*m.Source, error) {
	for _, i := range mockSourceDao.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (mockSourceDao *MockSourceDao) ListForRhcConnection(_ *int64, _, _ int, _ []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(mockSourceDao.RelatedSources))

	return mockSourceDao.RelatedSources, count, nil
}

func (mockSourceDao *MockSourceDao) DeleteCascade(id int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error) {
	var source *m.Source

	for _, src := range fixtures.TestSourceData {
		if src.ID == id {
			source = &src
		}
	}

	if source == nil {
		return nil, nil, nil, nil, nil, util.NewErrNotFound("source")
	}

	return fixtures.TestApplicationAuthenticationData, fixtures.TestApplicationData, fixtures.TestEndpointData, fixtures.TestRhcConnectionData, source, nil
}

func (mockSourceDao *MockSourceDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (mockSourceDao *MockSourceDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (mockSourceDao *MockSourceDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (mockSourceDao *MockSourceDao) Pause(_ int64) error {
	return nil
}

func (mockSourceDao *MockSourceDao) Unpause(_ int64) error {
	return nil
}
