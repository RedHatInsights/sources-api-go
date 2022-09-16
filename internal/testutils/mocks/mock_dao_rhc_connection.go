package mocks

import (
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

type MockRhcConnectionDao struct {
	RhcConnections        []m.RhcConnection
	RelatedRhcConnections []m.RhcConnection
}

func (mockRhcConnectionDao *MockRhcConnectionDao) List(_, _ int, _ []util.Filter) ([]m.RhcConnection, int64, error) {
	count := int64(len(mockRhcConnectionDao.RhcConnections))
	return mockRhcConnectionDao.RhcConnections, count, nil
}

func (mockRhcConnectionDao *MockRhcConnectionDao) GetById(id *int64) (*m.RhcConnection, error) {
	// The ".ToResponse" method of the RhcConnection expects to have at least one related source.
	source := []m.Source{
		{
			ID: 1,
		},
	}

	for _, rhcConnection := range mockRhcConnectionDao.RhcConnections {
		if rhcConnection.ID == *id {
			rhcConnection.Sources = source
			return &rhcConnection, nil
		}
	}

	return nil, util.NewErrNotFound("rhcConnection")
}

func (mockRhcConnectionDao *MockRhcConnectionDao) Create(rhcConnection *m.RhcConnection) (*m.RhcConnection, error) {
	// Check if in fixtures is a source with given source id
	var sourceExists bool
	for _, src := range fixtures.TestSourceData {
		if src.ID == rhcConnection.Sources[0].ID {
			sourceExists = true
		}
	}

	if !sourceExists {
		return nil, util.NewErrNotFound("source")
	}

	// Check if in fixtures exists same relation
	var relationExists bool
	for _, s := range fixtures.TestSourceRhcConnectionData {
		if s.SourceId == rhcConnection.Sources[0].ID {
			for _, r := range fixtures.TestRhcConnectionData {
				if s.RhcConnectionId == r.ID && r.RhcId == rhcConnection.RhcId {
					relationExists = true
				}
			}
		}
	}

	if relationExists {
		return nil, util.NewErrBadRequest("connection already exists")
	}

	return rhcConnection, nil
}

func (mockRhcConnectionDao *MockRhcConnectionDao) Update(rhcConnection *m.RhcConnection) error {
	for _, rhcTmp := range mockRhcConnectionDao.RhcConnections {
		if rhcTmp.ID == rhcConnection.ID {
			return nil
		}
	}

	return util.NewErrNotFound("rhcConnection")
}

func (mockRhcConnectionDao *MockRhcConnectionDao) Delete(id *int64) (*m.RhcConnection, error) {
	for _, rhcTmp := range mockRhcConnectionDao.RhcConnections {
		if rhcTmp.ID == *id {
			return &rhcTmp, nil
		}
	}

	return nil, util.NewErrNotFound("rhcConnection")
}

func (mockRhcConnectionDao *MockRhcConnectionDao) ListForSource(_ *int64, _, _ int, _ []util.Filter) ([]m.RhcConnection, int64, error) {
	count := int64(len(mockRhcConnectionDao.RelatedRhcConnections))

	return mockRhcConnectionDao.RelatedRhcConnections, count, nil
}
