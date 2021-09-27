package model

import (
	"regexp"
	"strconv"
	"testing"

	sourceConstants "github.com/RedHatInsights/sources-api-go/util/source"
)

var uuidRegex = regexp.MustCompile(`[a-f\d]{8}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{12}`)

// setUp returns a freshly created and valid SourceCreateRequest.
func setUp() SourceCreateRequest {
	name := "TestRequest"
	uid := "5"
	version := "10.5"
	imported := "true"
	sourceRef := "Source reference #5"
	sourceTypeId := "501"

	return SourceCreateRequest{
		Name:                &name,
		Uid:                 &uid,
		Version:             &version,
		Imported:            &imported,
		SourceRef:           &sourceRef,
		AppCreationWorkflow: sourceConstants.AccountAuth,
		AvailabilityStatus:  sourceConstants.Available,
		SourceTypeIDRaw:     &sourceTypeId,
	}

}

// TestValidRequest tests that a valid request doesn't report any errors when validated.
func TestValidRequest(t *testing.T) {
	request := setUp()

	err := request.Validate()
	if err != nil {
		t.Errorf("Request validation went wrong. No errors expected, got \"%s\"", err)
	}
}

// TestInvalidName tests that "Invalid name" errors are reported when validating the request.
func TestInvalidName(t *testing.T) {
	request := setUp()
	request.Name = nil

	err := request.Validate()
	if err == nil {
		t.Errorf("Name validation went wrong. Invalid name error expected, none gotten")
	}

	emptyName := ""
	request.Name = &emptyName

	if err == nil {
		t.Errorf("Name validation went wrong. Invalid name error expected, none gotten")
	}
}

// TestNonEmptyUuid tests that when a "valid" UUID is passed, it is not overwritten by the UUID generator.
func TestNonEmptyUuid(t *testing.T) {
	request := setUp()
	originalUuid := *request.Uid // we store the original uuid to do checks later

	err := request.Validate()
	if err != nil {
		t.Errorf("No errors are expected, got \"%s\"", err)
	}

	if *request.Uid != originalUuid {
		t.Errorf("Unexpected UUID change detected. Expected \"%s\", got \"%s\"", originalUuid, *request.Uid)
	}
}

// TestEmptyUuid tests that when an empty or a nil UUID is passed, a new one is generated.
func TestEmptyUuid(t *testing.T) {
	request := setUp()

	// Check that when passing a nil as a UUID, a new one is generated
	request.Uid = nil

	err := request.Validate()
	if err != nil {
		t.Errorf("No errors are expected, got \"%s\"", err)
	}

	if !uuidRegex.MatchString(*request.Uid) {
		t.Errorf("A generated UUID expected, got \"%s\"", *request.Uid)
	}

	// Check that when passing an empty string as a UUID, a new one is generated
	emptyUid := ""
	request.Uid = &emptyUid

	err = request.Validate()
	if err != nil {
		t.Errorf("No errors expected, got \"%s\"", err)
	}

	if !uuidRegex.MatchString(*request.Uid) {
		t.Errorf("A generated UUID expected, got \"%s\"", *request.Uid)
	}
}

// TestAppCreationWorkflowValues tests that the defined acceptable "AppCreationWorkflow" values are accepted. It also
// performs tests with invalid values to test the opposite.
func TestAppCreationWorkflowValues(t *testing.T) {
	request := setUp()

	// The request already has a valid value, but just in case we're going to test all the valid cases again
	var validValues = []string{
		sourceConstants.AccountAuth,
		sourceConstants.ManualConfig,
	}

	for _, validValue := range validValues {
		request.AppCreationWorkflow = validValue
		err := request.Validate()

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
		err := request.Validate()

		if err == nil {
			t.Errorf("Error expected when validating \"AppCreationWorkflow\", none gotten")
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
		sourceConstants.Available,
		sourceConstants.InProgress,
		sourceConstants.PartiallyAvailable,
		sourceConstants.Unavailable,
	}

	for _, validStatus := range validStatuses {
		request.AvailabilityStatus = validStatus

		err := request.Validate()
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

		err := request.Validate()
		if err == nil {
			t.Errorf("Error expected when validating \"AvailabilityStatus\", none gotten")
		}
	}
}

// TestSourceTypeIdParsing tests that when given an *int64 greater than 0, or a *string containing a number greater
// than 0, the parsing is done correctly.
func TestValidSourceTypeIdParsing(t *testing.T) {
	request := setUp()

	var validId int64 = 5
	request.SourceTypeIDRaw = &validId

	err := request.Validate()
	if err != nil {
		t.Errorf("No errors expected, got \"%s\"", err)
	}

	if validId != *request.SourceTypeID {
		t.Errorf(
			"Error when validating that the \"SourceTypeID\" is the same. Expected %d, got %d",
			validId,
			*request.SourceTypeID,
		)
	}

	validStringId := "519"
	validParsedId, err := strconv.ParseInt(validStringId, 10, 64)
	if err != nil {
		t.Errorf("Error converting string \"%d\" to int", validId)
	}

	request.SourceTypeIDRaw = &validStringId

	err = request.Validate()
	if err != nil {
		t.Errorf("No errors expected, got \"%s\"", err)
	}

	if validParsedId != *request.SourceTypeID {
		t.Errorf(
			"Error when validating that the \"SourceTypeID\" is the same. Expected %d, got %d",
			validId,
			*request.SourceTypeID,
		)
	}
}

// TestInvalidSourceTypeIdParsing tests that when passed an ID of an unexpected type, or if the given ID is less or
// equal to 0, the errors are correctly reported. What's being validated?
// - "0" *string
// - "50" string
// - 0 *int64
// - 50 int64
// - 50.2 *float64
func TestInvalidSourceTypeIdParsing(t *testing.T) {
	request := setUp()

	invalidString := "0"
	request.SourceTypeIDRaw = &invalidString

	err := request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing a string containing a number lower than 0, none gotten")
	}

	// The validator expects an *string, not a string
	invalidStringByValue := "50"
	request.SourceTypeIDRaw = invalidStringByValue

	err = request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing a string by value, none gotten")
	}

	var invalidInt int64 = 0
	request.SourceTypeIDRaw = &invalidInt

	err = request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing an int64 lower than 0, none gotten")
	}

	var invalidIntValue int64 = 50
	request.SourceTypeIDRaw = invalidIntValue

	err = request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing an int64 by value, none gotten")
	}

	invalidFloat := 50.2
	request.SourceTypeIDRaw = &invalidFloat

	err = request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing a float, none gotten")
	}

	invalidBoolean := false
	request.SourceTypeIDRaw = &invalidBoolean
	err = request.Validate()
	if err == nil {
		t.Errorf("Error expected when passing a bool, none gotten")
	}
}
