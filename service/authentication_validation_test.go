package service

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/model"
)

func TestIntId(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: 17,
		ResourceType:  "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}
}

func TestStringId(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidResourceType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "Thing",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestCapitalizeResource(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
		ResourceType:  "source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err != nil {
		t.Error(err)
	}

	if acr.ResourceType != "Source" {
		t.Errorf("Resource Type not sanitized")
	}
}

func TestMissingResourceType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceIDRaw: "17",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestMissingResourceID(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceType: "Source",
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestInvalidIdType(t *testing.T) {
	acr := model.AuthenticationCreateRequest{
		ResourceType:  "Source",
		ResourceIDRaw: struct{}{},
	}

	err := ValidateAuthenticationCreationRequest(&acr)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}
