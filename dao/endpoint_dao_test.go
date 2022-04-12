package dao

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestDeleteEndpoint tests that an endpoint gets correctly deleted, and its data returned.
func TestDeleteEndpoint(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

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

	DropSchema("delete")
}

// TestDeleteEndpointNotExists tests that when an endpoint that doesn't exist is tried to be deleted, an error is
// returned.
func TestDeleteEndpointNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("delete")

	endpointDao := GetEndpointDao(&fixtures.TestSourceData[0].TenantID)

	nonExistentId := int64(12345)
	_, err := endpointDao.Delete(&nonExistentId)

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf(`incorrect error returned. Want "%s", got "%s"`, util.ErrNotFoundEmpty, reflect.TypeOf(err))
	}

	DropSchema("delete")
}

// TestEndpointExists tests whether the function exists returns true when the given endpoint exists.
func TestEndpointExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	endpointDao := GetEndpointDao(&fixtures.TestTenantData[0].Id)

	got, err := endpointDao.Exists(fixtures.TestEndpointData[0].ID)
	if err != nil {
		t.Errorf(`unexpected error when checking that the endpoint exists: %s`, err)
	}

	if !got {
		t.Errorf(`the endpoint does exist but the "Exist" function returns otherwise. Want "true", got "%t"`, got)
	}

	DropSchema("exists")
}

// TestEndpointNotExists tests whether the function exists returns false when the given endpoint does not exist.
func TestEndpointNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("exists")

	endpointDao := GetEndpointDao(&fixtures.TestTenantData[0].Id)

	got, err := endpointDao.Exists(12345)
	if err != nil {
		t.Errorf(`unexpected error when checking that the endpoint exists: %s`, err)
	}

	if got {
		t.Errorf(`the endpoint doesn't exist but the "Exist" function returns otherwise. Want "false", got "%t"`, got)
	}

	DropSchema("exists")
}
