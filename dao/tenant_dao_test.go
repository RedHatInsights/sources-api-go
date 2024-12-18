package dao

import (
	"errors"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// TestGetOrCreateTenantIDEbsNumberCreate tests that when the EBS account number is not found, a new tenant is created.
func TestGetOrCreateTenantIDEbsNumberCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	accountNumber := "98765"
	identityStruct := identity.Identity{
		AccountNumber: accountNumber,
	}

	tenantDao := GetTenantDao()

	dbTenant, err := tenantDao.GetOrCreateTenant(&identityStruct)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	var tenant model.Tenant
	err = DB.
		Debug().
		Model(&model.Tenant{}).
		Where(`id = ?`, dbTenant.Id).
		First(&tenant).
		Error

	if err != nil {
		t.Errorf(`error fetching the tenant. Want nil error, got "%s"`, err)
	}

	want := accountNumber
	got := tenant.ExternalTenant

	if want != got {
		t.Errorf(`unexpected tenant fetched. Want EBS number "%s", got "%s"`, want, got)
	}

	DropSchema("tenant_tests")
}

// TestGetOrCreateTenantIDEbsNumberFind tests that when the EBS account number is found, the associated tenant id is
// returned.
func TestGetOrCreateTenantIDEbsNumberFind(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	identityStruct := identity.Identity{
		AccountNumber: fixtures.TestTenantData[0].ExternalTenant,
	}

	tenantDao := GetTenantDao()

	dbTenant, err := tenantDao.GetOrCreateTenant(&identityStruct)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	var tenant model.Tenant
	err = DB.
		Debug().
		Model(&model.Tenant{}).
		Where(`id = ?`, dbTenant.Id).
		First(&tenant).
		Error

	if err != nil {
		t.Errorf(`error fetching the tenant. Want nil error, got "%s"`, err)
	}

	want := fixtures.TestTenantData[0].ExternalTenant
	got := tenant.ExternalTenant

	if want != got {
		t.Errorf(`unexpected tenant fetched. Want EBS number "%s", got "%s"`, want, got)
	}

	DropSchema("tenant_tests")
}

// TestGetOrCreateTenantIDOrgIdCreate tests that when the OrgId is not found, a new tenant is created.
func TestGetOrCreateTenantIDOrgIdCreate(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	orgId := "1239875"
	identityStruct := identity.Identity{
		OrgID: orgId,
	}

	tenantDao := GetTenantDao()

	dbTenant, err := tenantDao.GetOrCreateTenant(&identityStruct)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	var tenant model.Tenant
	err = DB.
		Debug().
		Model(&model.Tenant{}).
		Where(`id = ?`, dbTenant.Id).
		First(&tenant).
		Error

	if err != nil {
		t.Errorf(`error fetching the tenant. Want nil error, got "%s"`, err)
	}

	want := orgId
	got := tenant.OrgID

	if want != got {
		t.Errorf(`unexpected tenant fetched. Want EBS number "%s", got "%s"`, want, got)
	}

	DropSchema("tenant_tests")
}

// TestGetOrCreateTenantIDOrgIdFind tests that when the OrgId is found, the associated tenant id is returned.
func TestGetOrCreateTenantIDOrgIdFind(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	identityStruct := identity.Identity{
		OrgID: fixtures.TestTenantData[0].OrgID,
	}

	tenantDao := GetTenantDao()

	dbTenant, err := tenantDao.GetOrCreateTenant(&identityStruct)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	var tenant model.Tenant
	err = DB.
		Debug().
		Model(&model.Tenant{}).
		Where(`id = ?`, dbTenant.Id).
		First(&tenant).
		Error

	if err != nil {
		t.Errorf(`error fetching the tenant. Want nil error, got "%s"`, err)
	}

	want := fixtures.TestTenantData[0].OrgID
	got := tenant.OrgID

	if want != got {
		t.Errorf(`unexpected tenant fetched. Want EBS number "%s", got "%s"`, want, got)
	}

	DropSchema("tenant_tests")
}

// TestTenantByIdentity tests that the function is able to fetch by either EBS account number or OrgId.
func TestTenantByIdentity(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	tenantDao := GetTenantDao()

	// Call the function under test by fetching by EBS account number.
	tenant, err := tenantDao.TenantByIdentity(&identity.Identity{
		AccountNumber: fixtures.TestTenantData[0].ExternalTenant,
	})
	if err != nil {
		t.Errorf(`unexpected error when fetching tenant: %s`, err)
	}

	{
		want := fixtures.TestTenantData[0].ExternalTenant
		got := tenant.ExternalTenant

		if want != got {
			t.Errorf(`incorrect tenant fetched. Want external tenant "%s", got "%s"`, want, got)
		}
	}

	// Call the function under test by fetching by orgId.
	tenant, err = tenantDao.TenantByIdentity(&identity.Identity{
		OrgID: fixtures.TestTenantData[0].OrgID,
	})
	if err != nil {
		t.Errorf(`unexpected error when fetching tenant: %s`, err)
	}

	{
		want := fixtures.TestTenantData[0].OrgID
		got := tenant.OrgID

		if want != got {
			t.Errorf(`incorrect tenant fetched. Want external tenant "%s", got "%s"`, want, got)
		}
	}

	DropSchema("tenant_tests")
}

// TestTenantByIdentityNotFound tests that a "not found" error is returned when the tenant is not found.
func TestTenantByIdentityNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	tenantDao := GetTenantDao()

	// Call the function under test by providing it an invalid account number.
	_, err := tenantDao.TenantByIdentity(&identity.Identity{
		AccountNumber: "invalid",
	})

	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf(`unexpected error recevied. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFound{}), reflect.TypeOf(err))
	}

	// Call the function under test by providing it an invalid orgId.
	_, err = tenantDao.TenantByIdentity(&identity.Identity{
		OrgID: "invalid",
	})

	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf(`unexpected error recevied. Want "%s", got "%s"`, reflect.TypeOf(util.ErrNotFound{}), reflect.TypeOf(err))
	}

	DropSchema("tenant_tests")
}

// TestCreateTenantNullEbsOrgId tests that when an empty string is received on the "AccountNumber" and "OrgId" identity
// struct, a "NULL" value is stored in the database. This is important because the unique indexes of the
// "external_tenant" and "org_id" columns don't consider "NULL"s as duplicates, but they do consider empty strings as
// duplicates.
func TestCreateTenantNullEbsOrgId(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("tenant_tests")

	tenantDao := GetTenantDao()

	// Try to insert a tenant without an "external_tenant" and "org_id" values.
	tenant, err := tenantDao.GetOrCreateTenant(&identity.Identity{})
	if err != nil {
		t.Errorf(`unexpected error when creating a tenant with a NULL EBS account number and OrgId: %s`, err)
	}

	// Fetch the created tenant. We need to use a "map[string]interface{}" because the tenant model doesn't use a
	// pointer value, and therefore the "NULL" value from the database would get mapped as an empty string.
	var createdTenant map[string]interface{}
	err = DB.
		Debug().
		Model(&model.Tenant{}).
		Where("id = ?", tenant.Id).
		Find(&createdTenant).
		Error

	if err != nil {
		t.Errorf(`error when trying to find the created tenant: %s`, err)
	}

	// Check if the "external_tenant" column is present.
	externalTenant, ok := createdTenant["external_tenant"]
	if !ok {
		t.Errorf(`could not find "external_tenant" column on a "get tenant by id" query: %s`, err)
	}

	// The expected value on the database should be "NULL".
	if externalTenant != nil {
		t.Errorf(`unexpected value returned. Want nil "external_tenant", got "%s"`, externalTenant)
	}

	// Check if the "org_id" column is present.
	orgId, ok := createdTenant["org_id"]
	if !ok {
		t.Errorf(`could not find "org_id" column on a "get tenant by id" query: %s`, err)
	}

	// The expected value on the database should be "NULL".
	if orgId != nil {
		t.Errorf(`unexpected value returned. Want nil "org_id", got "%s"`, orgId)
	}

	DropSchema("tenant_tests")
}
