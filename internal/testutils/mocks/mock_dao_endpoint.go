package mocks

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockEndpointDao struct {
	Endpoints []m.Endpoint
}

func (a *MockEndpointDao) SubCollectionList(primaryCollection interface{}, limit, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	// Return not found err when the source doesn't exist
	var sourceExist bool
	var sourceId int64
	switch object := primaryCollection.(type) {
	case m.Source:
		for _, source := range fixtures.TestSourceData {
			if object.ID == source.ID {
				sourceExist = true
				sourceId = object.ID
				break
			}
		}
	default:
		return nil, 0, fmt.Errorf("unexpected primary collection type")
	}

	if !sourceExist {
		return nil, 0, util.NewErrNotFound("source")
	}

	// Get list of endpoints for existing source
	var endpointsOut []m.Endpoint
	for _, e := range a.Endpoints {
		if e.SourceID == sourceId {
			endpointsOut = append(endpointsOut, e)
		}
	}

	return endpointsOut, int64(len(endpointsOut)), nil
}

func (a *MockEndpointDao) List(limit int, offset int, filters []util.Filter) ([]m.Endpoint, int64, error) {
	count := int64(len(a.Endpoints))
	return a.Endpoints, count, nil
}

func (a *MockEndpointDao) GetById(id *int64) (*m.Endpoint, error) {
	for _, app := range a.Endpoints {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("endpoint")
}

func (a *MockEndpointDao) Create(src *m.Endpoint) error {
	return nil
}

func (a *MockEndpointDao) Update(endpoint *m.Endpoint) error {
	if endpoint.ID == fixtures.TestEndpointData[0].ID {
		return nil
	}

	return util.NewErrNotFound("endpoint")
}

func (endpointDao *MockEndpointDao) Delete(id *int64) (*m.Endpoint, error) {
	for i, e := range endpointDao.Endpoints {
		if e.ID == *id {
			endpointDao.Endpoints = append(endpointDao.Endpoints[:i], endpointDao.Endpoints[i+1:]...)

			return &e, nil
		}
	}
	return nil, util.NewErrNotFound("endpoint")
}

func (m *MockEndpointDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (m *MockEndpointDao) CanEndpointBeSetAsDefaultForSource(sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) IsRoleUniqueForSource(role string, sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) SourceHasEndpoints(sourceId int64) bool {
	return true
}

func (m *MockEndpointDao) Exists(endpointId int64) (bool, error) {
	return true, nil
}

func (m *MockEndpointDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockEndpointDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (m *MockEndpointDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}
