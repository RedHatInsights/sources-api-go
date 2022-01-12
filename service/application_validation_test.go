package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
)

func TestValidApplicationRequest(t *testing.T) {
	appTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		SourceIDRaw:          "1",
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err != nil {
		t.Errorf("Got an error when there should not have been one: %v", err)
	}

	if req.ApplicationTypeID != 1 {
		t.Errorf("Failed to parse app type id, got %v, wanted 1", req.ApplicationTypeID)
	}

	if req.ApplicationTypeID != 1 {
		t.Errorf("Failed to parse app type id, got %v, wanted 1", req.ApplicationTypeID)
	}

	if req.SourceID != 1 {
		t.Errorf("Failed to parse source id, got %v, wanted 1", req.SourceID)
	}
}

func TestInvalidApplicationTypeForSource(t *testing.T) {
	appTypeDao = &dao.MockApplicationTypeDao{Compatible: false}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		SourceIDRaw:          "1",
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("Got no error when there should not have been one")
	}

	if !strings.Contains(err.Error(), "is not compatible") {
		t.Errorf("malformed error message: %v", err)
	}
}

func TestErrorCheckingApplicationType(t *testing.T) {
	appTypeDao = &dao.MockApplicationTypeDao{Compatible: false, CompatibleError: errors.New("boom!")}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		SourceIDRaw:          "1",
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("Got no error when there should not have been one")
	}

	if !strings.Contains(err.Error(), "failed to check") {
		t.Errorf("malformed error message: %v", err)
	}
}

func TestMissingApplicationTypeId(t *testing.T) {
	appTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:       []byte(`{"a good":"thing"`),
		SourceIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("No error when there should have been one")
	}
}

func TestMissingSourceId(t *testing.T) {
	appTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		Extra:                []byte(`{"a good":"thing"`),
		ApplicationTypeIDRaw: "1",
	}

	err := ValidateApplicationCreateRequest(&req)
	if err == nil {
		t.Errorf("No error when there should have been one")
	}
}
