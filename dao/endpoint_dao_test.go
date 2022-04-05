package dao

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/helpers"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestDeleteEndpoint tests that an endpoint gets correctly deleted, and its data returned.
func TestDeleteEndpoint(t *testing.T) {
	helpers.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	endpointDao := GetEndpointDao(&fixtures.TestSourceData[0].TenantID)

	endpoint := fixtures.TestEndpointData[0]
	// Set the ID to 0 to let GORM know it should insert a new endpoint and not update an existing one.
	endpoint.ID = 0
	// Set some data to compare the returned endpoint.
	host := "example.org"
	endpoint.Host = &host

	// Create the test endpoint.
	err := endpointDao.Create(&endpoint)
	if err != nil {
		t.Errorf("error creating endpoint: %s", err)
	}

	deletedEndpoint, err := endpointDao.Delete(&endpoint.ID)
	if err != nil {
		t.Errorf("error deleting an endpoint: %s", err)
	}

	{
		want := endpoint.ID
		got := deletedEndpoint.ID

		if want != got {
			t.Errorf(`incorrect endpoint deleted. Want id "%d", got "%d"`, want, got)
		}
	}

	{
		want := host
		got := deletedEndpoint.Host

		if want != *got {
			t.Errorf(`incorrect endpoint deleted. Want "%s" in the host field, got "%s"`, want, *got)
		}
	}

	DoneWithFixtures("delete")
}

// TestDeleteEndpointNotExists tests that when an endpoint that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteEndpointNotExists(t *testing.T) {
	helpers.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("delete")

	endpointDao := GetEndpointDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := endpointDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DoneWithFixtures("delete")
}
