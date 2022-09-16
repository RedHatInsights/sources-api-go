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

func (mockEndpointDao *MockEndpointDao) SubCollectionList(primaryCollection interface{}, _, _ int, _ []util.Filter) ([]m.Endpoint, int64, error) {
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
	for _, e := range mockEndpointDao.Endpoints {
		if e.SourceID == sourceId {
			endpointsOut = append(endpointsOut, e)
		}
	}

	return endpointsOut, int64(len(endpointsOut)), nil
}

func (mockEndpointDao *MockEndpointDao) List(_ int, _ int, _ []util.Filter) ([]m.Endpoint, int64, error) {
	count := int64(len(mockEndpointDao.Endpoints))
	return mockEndpointDao.Endpoints, count, nil
}

func (mockEndpointDao *MockEndpointDao) GetById(id *int64) (*m.Endpoint, error) {
	for _, app := range mockEndpointDao.Endpoints {
		if app.ID == *id {
			return &app, nil
		}
	}

	return nil, util.NewErrNotFound("endpoint")
}

func (mockEndpointDao *MockEndpointDao) Create(_ *m.Endpoint) error {
	return nil
}

func (mockEndpointDao *MockEndpointDao) Update(endpoint *m.Endpoint) error {
	if endpoint.ID == fixtures.TestEndpointData[0].ID {
		return nil
	}

	return util.NewErrNotFound("endpoint")
}

func (mockEndpointDao *MockEndpointDao) Delete(id *int64) (*m.Endpoint, error) {
	for i, e := range mockEndpointDao.Endpoints {
		if e.ID == *id {
			mockEndpointDao.Endpoints = append(mockEndpointDao.Endpoints[:i], mockEndpointDao.Endpoints[i+1:]...)

			return &e, nil
		}
	}
	return nil, util.NewErrNotFound("endpoint")
}

func (mockEndpointDao *MockEndpointDao) Tenant() *int64 {
	tenant := int64(1)
	return &tenant
}

func (mockEndpointDao *MockEndpointDao) CanEndpointBeSetAsDefaultForSource(_ int64) bool {
	return true
}

func (mockEndpointDao *MockEndpointDao) IsRoleUniqueForSource(_ string, _ int64) bool {
	return true
}

func (mockEndpointDao *MockEndpointDao) SourceHasEndpoints(_ int64) bool {
	return true
}

func (mockEndpointDao *MockEndpointDao) Exists(_ int64) (bool, error) {
	return true, nil
}

func (mockEndpointDao *MockEndpointDao) BulkMessage(_ util.Resource) (map[string]interface{}, error) {
	return nil, nil
}

func (mockEndpointDao *MockEndpointDao) FetchAndUpdateBy(_ util.Resource, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (mockEndpointDao *MockEndpointDao) ToEventJSON(_ util.Resource) ([]byte, error) {
	return nil, nil
}
