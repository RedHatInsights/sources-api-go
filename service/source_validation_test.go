package service

import (
	"math"
	"regexp"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/model"
)

var uuidRegex = regexp.MustCompile(`[a-f\d]{8}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{12}`)

// setUp returns a freshly created and valid SourceCreateRequest.
func setUp() model.SourceCreateRequest {
	name := "TestRequest"
	version := "10.5"
	imported := "true"
	sourceRef := "Source reference #5"
	sourceTypeId := "501"

	return model.SourceCreateRequest{
		Name:                &name,
		Version:             &version,
		Imported:            &imported,
		SourceRef:           &sourceRef,
		AppCreationWorkflow: model.AccountAuth,
		AvailabilityStatus:  model.Available,
		SourceTypeIDRaw:     &sourceTypeId,
	}

}

// TestValidRequest tests that a valid request doesn't report any errors when validated.
func TestValidRequest(t *testing.T) {
	request := setUp()

	err := ValidateSourceCreationRequest(sourceDao, &request)
	if err != nil {
		t.Errorf("Request validation went wrong. No errors expected, got \"%s\"", err)
	}
}

// TestInvalidName tests that "Invalid name" errors are reported when validating the request.
func TestInvalidName(t *testing.T) {
	request := setUp()
	request.Name = nil

	err := ValidateSourceCreationRequest(sourceDao, &request)
	if err == nil {
		t.Errorf("Name validation went wrong. Invalid name error expected, none gotten")
	}

	emptyName := ""
	request.Name = &emptyName

	if err == nil {
		t.Errorf("Name validation went wrong. Invalid name error expected, none gotten")
	}
}

// TestInvalidDuplicatedNameInTenant tests that the validation fails if the given source's name is not unique in the
// tenant. For this purpose it creates a new source in the database and then deletes it instead of using the existing
// fixture that is inserted in the main function. The reason is that it is easier to control this new fixture here
// than having to track the name of the previously inserted fixture, or exporting it to variable or whatever.
func TestInvalidDuplicatedNameInTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	sourceName := "Source350"
	sourceUid := "abcde-fghijk"
	newSource := model.Source{ID: 350, Name: sourceName, SourceTypeID: 1, TenantID: 1, Uid: &sourceUid}
	err := dao.DB.
		Debug().
		Create(&newSource).
		Error

	if err != nil {
		t.Errorf(`could not create the source fixture for the test: %s`, err)
	}

	request := setUp()
	request.Name = &sourceName

	err = ValidateSourceCreationRequest(sourceDao, &request)

	if err == nil {
		t.Errorf("Error expected, got none")
	}

	if err.Error() != "name already exists in tenant" {
		t.Errorf("want %#v, got %#v", "name already exists in tenant", err.Error())
	}

	dao.DB.Delete(newSource)
}

// TestUuidGeneration tests that UUIDs are correctly generated when validating a new source.
func TestUuidGeneration(t *testing.T) {
	request := setUp()

	for i := 0; i < 5; i++ {
		err := ValidateSourceCreationRequest(sourceDao, &request)
		if err != nil {
			t.Errorf("No errors are expected, got \"%s\"", err)
		}

		if !uuidRegex.MatchString(*request.Uid) {
			t.Errorf("A generated UUID expected, got \"%s\"", *request.Uid)
		}
	}
}

// TestAppCreationWorkflowValues tests that the defined acceptable "AppCreationWorkflow" values are accepted. It also
// performs tests with invalid values to test if the default value is correctly set.
func TestAppCreationWorkflowValues(t *testing.T) {
	request := setUp()

	// The request already has a valid value, but just in case we're going to test all the valid cases again
	var validValues = []string{
		model.AccountAuth,
		model.ManualConfig,
	}

	for _, validValue := range validValues {
		request.AppCreationWorkflow = validValue
		err := ValidateSourceCreationRequest(sourceDao, &request)

		if err != nil {
			t.Errorf("No errors expected, got \"%s\"", err)
		}
	}

	var invalidValues = []string{
		"",
		"test",
		"123123",
		"hello",
		"world",
	}

	for _, invalidValue := range invalidValues {
		request.AppCreationWorkflow = invalidValue
		err := ValidateSourceCreationRequest(sourceDao, &request)

		if err != nil {
			t.Errorf("No errors expected, got %s", err)
		}

		if request.AppCreationWorkflow != model.ManualConfig {
			t.Errorf("want %s, got %s", model.ManualConfig, request.AppCreationWorkflow)
		}
	}
}

// TestAvailabilityStatusValues tests that the validation function does not return an error if an acceptable valid
// status is specified. It also tests with invalid statuses to check the opposite.
func TestAvailabilityStatusValues(t *testing.T) {
	request := setUp()

	// The request already has a valid status, but we're testing all the values just in case
	var validStatuses = []string{
		"",
		model.Available,
		model.InProgress,
		model.PartiallyAvailable,
		model.Unavailable,
	}

	for _, validStatus := range validStatuses {
		request.AvailabilityStatus = validStatus

		err := ValidateSourceCreationRequest(sourceDao, &request)
		if err != nil {
			t.Errorf("No errors expected, got \"%s\"", err)
		}
	}

	var invalidStatuses = []string{
		"test",
		"availalable",
		"progressIn",
		"hello",
		"warld",
	}

	for _, invalidStatus := range invalidStatuses {
		request.AvailabilityStatus = invalidStatus

		err := ValidateSourceCreationRequest(sourceDao, &request)
		if err == nil {
			t.Errorf("Error expected when validating \"AvailabilityStatus\", none gotten")
		}
	}
}

// TestSourceTypeIdLowerOne tests that when given a SourceTypeID lower than one, the Validate function returns an
// error
func TestSourceTypeIdLowerOne(t *testing.T) {
	request := setUp()

	var pointerNegativeInt int64 = -10
	var pointerNegativeFloat float64 = -1921
	var pointerNegativeString = "-12345"

	lowerZero := []struct {
		value interface{}
	}{
		{int64(-5)},
		{int64(0)},
		{&pointerNegativeInt},
		{int64(math.MinInt64)},
		{0.9999999999999},
		{-1123.12},
		{&pointerNegativeFloat},
		{"0"},
		{"-9"},
		{&pointerNegativeString},
	}

	for _, tt := range lowerZero {
		request.SourceTypeIDRaw = tt.value

		err := ValidateSourceCreationRequest(sourceDao, &request)

		if err == nil {
			t.Errorf("Error expected, got none")
		}

		if err.Error() != "source type id must be greater than 0" {
			t.Errorf("got \"%s\", want \"%s\"", err.Error(), "source type id must be greater than 0")
		}
	}
}

// TestInvalidSourceTypeIdFormat tests that upon receiving a source type id in an incorrect format, the validate
// function reports an error
func TestInvalidSourceTypeIdFormat(t *testing.T) {
	request := setUp()

	invalidTypes := []struct {
		value interface{}
	}{
		{true},
		{false},
		{"-0.9"},
		{"0.5"},
		{complex(14, 3)},
		{'5'},
	}

	for _, tt := range invalidTypes {
		request.SourceTypeIDRaw = tt.value
		err := ValidateSourceCreationRequest(sourceDao, &request)

		if err == nil {
			t.Errorf("Error expected, got none")
		}

		if err.Error() != "the source type id is not valid" {
			t.Errorf("got \"%s\", want \"%s\"", err.Error(), "the source type id is not valid")
		}
	}
}
