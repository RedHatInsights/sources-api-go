package middleware

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// mockTenantDao is a mock which will help us mocking the "GetOrCreateTenant" function.
type mockTenantDao struct {
	TenantId      int64
	AccountNumber string
	OrgId         string
}

func (mtd mockTenantDao) GetOrCreateTenant(_ *identity.Identity) (*m.Tenant, error) {
	tenant := m.Tenant{
		Id:             mtd.TenantId,
		ExternalTenant: mtd.AccountNumber,
		OrgID:          mtd.OrgId,
	}

	return &tenant, nil
}

// GetUntranslatedTenants is unimplemented since we are not using it in the tests.
func (_ mockTenantDao) GetUntranslatedTenants() ([]m.Tenant, error) {
	return nil, nil
}

// TenantByIdentity is unimplemented since we are not using it in the tests.
func (_ mockTenantDao) TenantByIdentity(_ *identity.Identity) (*m.Tenant, error) {
	return nil, nil
}

// GetUntranslatedTenants is unimplemented since we are not using it in the tests.
func (_ mockTenantDao) TranslateTenants() (int64, uint64, uint64, []m.TenantTranslation, error) {
	return 0, 0, 0, nil, nil
}

// TestTenancySetsAllTenancyVariables tests that even if we simply receive one of the two "EBS account number" and
// "OrgId" headers, the tenancy middleware will set whatever tenancy contents we have from the database in the context.
func TestTenancySetsAllTenancyVariables(t *testing.T) {
	// Set up the data for the mock.
	tenantId := int64(50)
	accountNumber := "12345"
	orgId := "organization-id"

	// Mock the "GetTenantDao" function, which will return a tenant with the above contents.
	mtd := mockTenantDao{TenantId: tenantId, AccountNumber: accountNumber, OrgId: orgId}
	dao.GetTenantDao = func() dao.TenantDao { return mtd }

	// Create a dummy request context to be able to run the middlewares.
	c, _ := request.CreateTestContext(http.MethodGet, "/", nil, nil)

	// Prepare the headers we will be testing this with. We will try with one at a time.
	testHeaders := map[string]string{
		headers.ACCOUNT_NUMBER: accountNumber,
		headers.ORGID:          orgId,
	}

	for key, value := range testHeaders {
		// Set one of the headers.
		c.Request().Header.Add(key, value)

		// Prepare the middlewares and execute them.
		handlers := ParseHeaders(Tenancy(func(context echo.Context) error { return nil }))
		err := handlers(c)
		if err != nil {
			t.Errorf(`unexpected error when running the "ParseHeaders" and "Tenancy" middlewares: %s`, err)
		}

		{
			id, ok := c.Get(headers.PARSED_IDENTITY).(*identity.XRHID)
			if !ok {
				t.Errorf("unable to correctly type cast the parsed identity from the context")
			}

			{
				want := orgId
				got := id.Identity.OrgID
				if want != got {
					t.Errorf(`the identity struct does not have the OrgId value from the database. Want "%s", got "%s"`, want, got)
				}
			}

			{
				want := accountNumber
				got := id.Identity.AccountNumber
				if want != got {
					t.Errorf(`the identity struct does not have the AccountNUmber value from the database. Want "%s", got "%s"`, want, got)
				}
			}
		}

		{
			want := accountNumber
			got := c.Get(headers.ACCOUNT_NUMBER)

			if want != got {
				t.Errorf(`"%s" header set with value "%s". Invalid account number set when going through the "ParseHeaders" and "Tenancy" middlewares. Want "%s", got "%s"`, key, value, want, got)
			}
		}

		{
			want := orgId
			got := c.Get(headers.ORGID)

			if want != got {
				t.Errorf(`"%s" header set with value "%s. Invalid OrgId set when going through the "ParseHeaders" and "Tenancy" middlewares. Want "%s", got "%s"`, key, value, want, got)
			}
		}

		{
			want := tenantId
			got := c.Get(headers.TENANTID)

			if want != got {
				t.Errorf(`"%s" header set with value "%s. Invalid tenant id set when going through the "ParseHeaders" and "Tenancy" middlewares. Want "%d", got "%v"`, key, value, want, got)
			}
		}
	}
}
