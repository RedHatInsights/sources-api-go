package middleware

import (
	"errors"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var permCheckOrElse204 = PermissionCheck(func(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
})

func TestRbacDisabled(t *testing.T) {
	bypassRbac = true

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}

	bypassRbac = false
}

func TestPSKMatches(t *testing.T) {
	psks = []string{"1234"}
	if pskMatches("1234") != true {
		t.Errorf("psk didn't match when it should have")
	}

	if pskMatches("12345") == true {
		t.Errorf("psk matched when it should not have")
	}
}

func TestGoodPSK(t *testing.T) {
	psks = []string{"1234"}
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

func TestBadPSK(t *testing.T) {
	psks = []string{"abcdef"}
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{h.PSK: "1234"},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 401 {
		t.Errorf("%v was returned instead of %v", rec.Code, 401)
	}
}

func TestNoPSK(t *testing.T) {
	psks = []string{"abcdef"}
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 401 {
		t.Errorf("%v was returned instead of %v", rec.Code, 401)
	}
}

func TestSystemClusterID(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						ClusterId: "test_cluster",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

func TestSystemCN(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

func TestSystemPatch(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 405 {
		t.Errorf("%v was returned instead of %v", rec.Code, 405)
	}
}

func TestSystemDelete(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 405 {
		t.Errorf("%v was returned instead of %v", rec.Code, 405)
	}
}

func TestSystemDeleteSource(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/sources/1235",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

func TestSystemDeleteSourceVersioned(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/1235",
		nil,
		map[string]interface{}{
			"x-rh-identity": "dummy",
			"identity": &identity.XRHID{
				Identity: identity.Identity{
					System: identity.System{
						CommonName: "test_cert",
					},
				},
			},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

// yay dummy structs!
type dummyRbac struct {
	access bool
	blowup bool
}

func (d dummyRbac) Allowed(_ string) (bool, error) {
	if d.blowup {
		return false, errors.New("kablooey!")
	}

	return d.access, nil
}

func TestRbacWithAccess(t *testing.T) {
	rbacClient = dummyRbac{access: true}

	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			"identity":      &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 204)
	}
}

func TestRbacWithoutAccess(t *testing.T) {
	rbacClient = dummyRbac{access: false}

	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			"identity":      &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	err := permCheckOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 401 {
		t.Errorf("%v was returned instead of %v", rec.Code, 401)
	}
}

func TestRbacNoConnection(t *testing.T) {
	rbacClient = dummyRbac{access: false, blowup: true}

	c, _ := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{
			"x-rh-identity": "a wild xrhid - i mean eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJtaWdyYXRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwiaHlicmlkX2Nsb3VkIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1Z",
			"identity":      &identity.XRHID{Identity: identity.Identity{}},
		},
	)

	err := permCheckOrElse204(c)

	if err == nil {
		t.Errorf("no error was returned when we were expecting one!")
	}
}
