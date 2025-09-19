package middleware

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// mockedRbacResponse defines the response that we will get from the RBAC client.
type mockedRbacResponse struct {
	AllowedResponse bool
	ErrorResponse   error
}

// mockRbacClient helps us mock RBAC responses.
type mockRbacClient struct {
	mockedRbacResponse
}

func (m *mockRbacClient) Allowed(string) (bool, error) {
	if m.ErrorResponse != nil {
		return false, m.ErrorResponse
	}

	return m.AllowedResponse, nil
}

// setUpMiddleware sets up a "PermissionCheck" middleware with the given arguments. It also sets the middleware up so
// that if no errors occur, a "204 — No content" response is returned.
func setUpMiddleware(bypassRbac bool, allowedPsks []string, authorizedJWTSubjects []config.AuthorizedJWTSubject, rbacResponse mockedRbacResponse) echo.HandlerFunc {
	mockedRbacClient := mockRbacClient{mockedRbacResponse: rbacResponse}

	middleware := PermissionCheck(bypassRbac, allowedPsks, authorizedJWTSubjects, &mockedRbacClient)

	return middleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
}

func TestRbacDisabled(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{},
	)

	middleware := setUpMiddleware(true, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestPSKMatches(t *testing.T) {
	allowedPsks := []string{"1234"}

	if pskMatches(allowedPsks, "1234") != true {
		t.Errorf("psk didn't match when it should have")
	}

	if pskMatches(allowedPsks, "12345") == true {
		t.Errorf("psk matched when it should not have")
	}
}

func TestGoodPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	middleware := setUpMiddleware(true, []string{"1234"}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestBadPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	middleware := setUpMiddleware(false, []string{"abcdef"}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestNoPSK(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{},
	)

	middleware := setUpMiddleware(false, []string{"abcdef"}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestSystemClusterID(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						ClusterId: "test_cluster",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(true, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestSystemCN(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(true, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestSystemPatch(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSystemDelete(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestSystemDeleteSource(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/sources/1235",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestSystemDeleteSourceVersioned(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/1235",
		nil,
		map[string]interface{}{
			h.XRHID: "dummy",
			h.ParsedIdentity: &identity.XRHID{
				Identity: identity.Identity{
					System: &identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestRbacWithAccess(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{AllowedResponse: true})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestRbacWithoutAccess(t *testing.T) {
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{AllowedResponse: false})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestRbacNoConnection(t *testing.T) {
	c, _ := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			h.XRHID:          "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			h.ParsedIdentity: &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{ErrorResponse: fmt.Errorf("unable to connect to rbac")})

	err := middleware(c)
	if err == nil {
		t.Errorf("no error was returned when we were expecting one!")
	}
}

func TestJWTClaimsMatches(t *testing.T) {
	authorizedJWTSubjects := []config.AuthorizedJWTSubject{
		{Issuer: "https://example.com", Subject: "test-subject"},
	}

	if jwtClaimsMatches(authorizedJWTSubjects, "https://example.com", "test-subject") != true {
		t.Errorf("jwt claims didn't match when they should have")
	}

	if jwtClaimsMatches(authorizedJWTSubjects, "https://wrong.com", "test-subject") == true {
		t.Errorf("jwt claims matched wrong issuer when they should not have")
	}

	if jwtClaimsMatches(authorizedJWTSubjects, "https://example.com", "wrong-subject") == true {
		t.Errorf("jwt claims matched wrong subject when they should not have")
	}
}

func TestGoodJWTSubject(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.JWTIssuer:  "https://example.com",
			h.JWTSubject: "test-subject",
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{{Issuer: "https://example.com", Subject: "test-subject"}}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}

func TestBadJWTSubject(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			h.JWTIssuer:  "https://example.com",
			h.JWTSubject: "wrong-subject",
		},
	)

	middleware := setUpMiddleware(false, []string{}, []config.AuthorizedJWTSubject{{Issuer: "https://example.com", Subject: "test-subject"}}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestJWTSubjectBypassRbac(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.JWTSubject: "any-subject"},
	)

	middleware := setUpMiddleware(true, []string{}, []config.AuthorizedJWTSubject{}, mockedRbacResponse{})

	err := middleware(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("%v was returned instead of %v", rec.Code, http.StatusNoContent)
	}
}
