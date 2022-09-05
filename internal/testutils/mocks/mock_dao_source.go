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

func (src *MockSourceDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
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
		for index, source := range src.Sources {
			if source.SourceTypeID == object.Id {
				sources = append(sources, src.Sources[index])
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

func (src *MockSourceDao) List(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(src.Sources))
	return src.Sources, count, nil
}

func (src *MockSourceDao) ListInternal(limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(src.Sources))
	return src.Sources, count, nil
}

func (src *MockSourceDao) GetById(id *int64) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (src *MockSourceDao) Create(s *m.Source) error {
	src.Sources = append(src.Sources, *s)
	return nil
}

func (src *MockSourceDao) Update(s *m.Source) error {
	return nil
}

func (src *MockSourceDao) Delete(id *int64) (*m.Source, error) {
	for i, source := range src.Sources {
		if source.ID == *id {
			src.Sources = append(src.Sources[:i], src.Sources[i+1:]...)

			return &source, nil
		}
	}
	return nil, util.NewErrNotFound("source")
}

func (src *MockSourceDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (src *MockSourceDao) User() *int64 {
	user := int64(1)
	return &user
}

func (src *MockSourceDao) Exists(sourceId int64) (bool, error) {
	for _, source := range src.Sources {
		if source.ID == sourceId {
			return true, nil
		}
	}

	return false, nil
}

// NameExistsInCurrentTenant returns always false because it's the safe default in case the request gets validated
// in the tests.
func (src *MockSourceDao) NameExistsInCurrentTenant(name string) bool {
	return false
}

func (src *MockSourceDao) IsSuperkey(id int64) bool {
	return false
}

func (src *MockSourceDao) GetByIdWithPreload(id *int64, preloads ...string) (*m.Source, error) {
	for _, i := range src.Sources {
		if i.ID == *id {
			return &i, nil
		}
	}

	return nil, util.NewErrNotFound("source")
}

func (m *MockSourceDao) ListForRhcConnection(id *int64, limit, offset int, filters []util.Filter) ([]m.Source, int64, error) {
	count := int64(len(m.RelatedSources))

	return m.RelatedSources, count, nil
}

func (msd *MockSourceDao) DeleteCascade(id int64) ([]m.ApplicationAuthentication, []m.Application, []m.Endpoint, []m.RhcConnection, *m.Source, error) {
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

func (m *MockSourceDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockSourceDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockSourceDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}

func (s *MockSourceDao) Pause(_ int64) error {
	return nil
}

func (s *MockSourceDao) Unpause(_ int64) error {
	return nil
}
