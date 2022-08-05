package service

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func setUpEndpointCreateRequest() model.EndpointCreateRequest {
	receptorNode := "receptorNode"
	scheme := "https"
	port := 443
	verifySsl := true
	certificateAuthority := "letsEncrypt"
	sourceId := strconv.FormatInt(fixtures.TestSourceData[0].ID, 10)

	return model.EndpointCreateRequest{
		Default:              false,
		ReceptorNode:         &receptorNode,
		Role:                 "role",
		Scheme:               &scheme,
		Host:                 "example.com",
		Port:                 &port,
		Path:                 "/example",
		VerifySsl:            &verifySsl,
		CertificateAuthority: &certificateAuthority,
		AvailabilityStatus:   model.Available,
		SourceIDRaw:          sourceId,
	}
}

// setUpEndpointEditRequest returns a valid "EndpointEditRequest" and an associated source ID to be able to validate it.
func setUpEndpointEditRequest() (model.EndpointEditRequest, int64) {
	defaultValue := false
	receptorNode := "receptorNode"
	role := "role"
	scheme := "https"
	host := "example.com"
	port := 443
	path := "/example"
	verifySsl := true
	certificateAuthority := "letsEncrypt"

	return model.EndpointEditRequest{
			Default:              &defaultValue,
			ReceptorNode:         &receptorNode,
			Role:                 &role,
			Scheme:               &scheme,
			Host:                 &host,
			Port:                 &port,
			Path:                 &path,
			VerifySsl:            &verifySsl,
			CertificateAuthority: &certificateAuthority,
			AvailabilityStatus:   util.StringRef(model.Available),
		},
		fixtures.TestSourceData[0].ID
}

// TestValidateEndpointCreateRequest tests that when a proper EndpointCreateRequest is given, the validator doesn't
// complain.
func TestValidateEndpointCreateRequest(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("want no errors, got '%s'", err)
	}
}

// TestInvalidSourceIdFormat tests if an error is returned when an invalid format or string is provided for the source
// id.
func TestInvalidSourceIdFormat(t *testing.T) {
	ecr := setUpEndpointCreateRequest()

	ecr.SourceIDRaw = "hello world"

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "the provided source ID is not valid"
	if err.Error() != want {
		t.Errorf("want '%s', got '%s'", want, err)
	}
}

// TestInvalidSourceId tests if an error is returned when providing a source id lower than one.
func TestInvalidSourceId(t *testing.T) {
	ecr := setUpEndpointCreateRequest()

	ecr.SourceIDRaw = "0"

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "invalid source id"
	if err.Error() != want {
		t.Errorf("want '%s', got '%s'", want, err)
	}
}

// TestDefaultEndpointAlreadyExists tests if an error is returned when the provided endpoint is marked as default when
// the also provided source already has a default one.
func TestDefaultEndpointAlreadyExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	ecr.Default = true

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "a default endpoint already exists for the provided source"
	if err.Error() != want {
		t.Errorf("want '%s' error, got '%s'", want, err)
	}
}

// TestDefaultIsSetBecauseSourceHasNoEndpoints tests if the endpoint is defaulted when the provided source  has no
// endpoints.
func TestDefaultIsSetBecauseSourceHasNoEndpoints(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	ecr.SourceIDRaw = "12345"

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("want no errors, got '%s'", err)
	}

	if ecr.Default != true {
		t.Error("want endpoint marked as default, got regular endpoint instead")
	}
}

// TestNonUniqueRole tests if an error is returned when a role already exists for a source.
func TestNonUniqueRole(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	role := "myRole"
	newEndpoint := fixtures.TestEndpointData[0]
	newEndpoint.ID = 0 // set it as zero to avoid hitting
	newEndpoint.Role = &role

	err := endpointDao.Create(&newEndpoint)
	if err != nil {
		t.Error("could not insert new endpoint for the test")
	}

	ecr := setUpEndpointCreateRequest()
	ecr.Role = role

	err = ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "the role already exists for the given source"
	if err.Error() != want {
		t.Errorf("want '%s', got %s", want, err)
	}
}

// TestSchemeGetsDefaulted tests if the scheme gets properly defaulted when an invalid or missing scheme is provided.
func TestSchemeGetsDefaulted(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	ecr.Scheme = nil

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)

	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}

	if *ecr.Scheme != defaultScheme {
		t.Errorf("want '%s', got '%s", defaultScheme, *ecr.Scheme)
	}

	invalidScheme := "/invalid"
	ecr.Scheme = &invalidScheme

	err = ValidateEndpointCreateRequest(endpointDao, &ecr)

	if err != nil {
		t.Errorf("want no errors, got %s", err)
	}

	if *ecr.Scheme != defaultScheme {
		t.Errorf("want '%s', got '%s", defaultScheme, *ecr.Scheme)
	}
}

// TestEmptyHost tests if no error is returned even when an empty host is given.
func TestEmptyHost(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	ecr.Host = ""

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("want no error, got '%s'", err)
	}
}

// TestHostFqdnTooLong tests if an error is returned when a host which is too long is given.
func TestHostFqdnTooLong(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	// The "longHostname" variable holds a 256 char hostname
	ecr.Host = `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaa
	`

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := "the provided host is longer than 255 characters"
	if err.Error() != want {
		t.Errorf("want '%s' got '%s'", want, err)
	}
}

// TestValidHosts tests if the validation succeeds when valid hosts are given.
func TestValidHosts(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testValues := []string{
		"redhat.com",
		"elpmaxe.example.com",
		"exa.mp-le.com",
		"123456789.org",
		"123ab-ba321.org",
		"a.org",
		"5.com",
		"example.xn--whatever",
	}

	ecr := setUpEndpointCreateRequest()

	for _, tt := range testValues {
		ecr.Host = tt

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err != nil {
			t.Errorf("want no errors, got '%s' for %#v", err, tt)
		}
	}
}

// TestInvalidHosts tests if an error is returned on invalid hosts.
func TestInvalidHosts(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testValues := []string{
		"-example.com",
		"example-.com",
		"example.xn--",
		"example.--xn",
	}

	ecr := setUpEndpointCreateRequest()

	want := "the provided host is not valid"
	for _, tt := range testValues {
		ecr.Host = tt

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err == nil {
			t.Errorf("want '%s', got no error for %#v", want, tt)
		}
	}
}

// TestLabelNamesTooLong tests if an error is returned when the provided labels are longer than permitted.
func TestLabelNamesTooLong(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	// The first label is 64 characters long
	ecr.Host = `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeffffffffffgggg.example.org`

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err == nil {
		t.Error("want error, got none")
	}

	want := fmt.Sprintf("the label '%s' is greater than %d characters", "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeffffffffffgggg", maxLabelLength)
	if err.Error() != want {
		t.Errorf("want '%s', got %s", want, err)
	}
}

// TestDefaultingPortWhenMissingOrLessThanZero test if the port is given a default value if it's missing or it has been
// given a value equal or lower than zero.
func TestDefaultingPortWhenMissingOrLessThanZero(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	minusOne := -1
	zero := 0
	testValues := []struct {
		value *int
	}{
		{nil},
		{&minusOne},
		{&zero},
	}

	for _, tt := range testValues {
		ecr.Port = tt.value
		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err != nil {
			t.Errorf("want no error, got %s", err)
		}

		if *ecr.Port != defaultPort {
			t.Errorf("want %d, got %d", defaultPort, *ecr.Port)
		}
	}
}

// TestPortLargeValue tests if an error is returned when a port that is greater than the maximum allowed port is given.
func TestPortLargeValue(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	largePort := 999999
	ecr.Port = &largePort

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)

	if err == nil {
		t.Error("want error, got none")
	}

	want := "invalid port number"
	if err.Error() != want {
		t.Errorf("want '%s', got '%s'", want, err)
	}
}

// TestDefaultVerifySsl tests if the default value is set when no "VerifySSL" value is provided.
func TestDefaultVerifySsl(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	ecr.VerifySsl = nil

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("want no error, got '%s'", err)
	}

	if *ecr.VerifySsl != defaultVerifySsl {
		t.Errorf("want %t, got %t", defaultVerifySsl, *ecr.VerifySsl)
	}
}

// TestEmptyCertificateAuthority tests if an error is returned when ssl verification is turned on but no certificate
// authority is given.
func TestEmptyCertificateAuthority(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	emptyString := ""
	testValues := []struct {
		value *string
	}{
		{nil},
		{&emptyString},
	}

	want := "the certificate authority cannot be empty"
	for _, tt := range testValues {
		ecr.CertificateAuthority = tt.value

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)

		if err.Error() != want {
			t.Errorf("want '%s', got '%s'", want, err)
		}
	}
}

// TestValidAvailabilityStatuses tests if no error is returned when valid availability statuses are given.
func TestValidAvailabilityStatuses(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	testValues := []string{model.Available, model.Unavailable}
	for _, tt := range testValues {
		ecr.AvailabilityStatus = tt

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err != nil {
			t.Errorf("want no error, got '%s'", err)
		}
	}
}

// TestDefaultAvailabilityStatus tests that a default value is set for the availability status when it comes empty.
func TestDefaultAvailabilityStatus(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()
	ecr.AvailabilityStatus = ""

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("unexpected error when checking for a default value on the availability status member of an endpoint: %s", err)
	}

	want := model.InProgress
	got := ecr.AvailabilityStatus

	if want != got {
		t.Errorf(`unexpected default value for an endpoint when the availability status comes empty. Want "%s", got "%s"`, want, got)
	}

}

// TestInvalidAvailabilityStatuses tests if an error is returned when passing invalid availability statuses.
func TestInvalidAvailabilityStatuses(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	ecr := setUpEndpointCreateRequest()

	invalidValues := []string{"hello", "world", "almost", "passes", "validation"}
	want := "invalid availability status"
	for _, tt := range invalidValues {
		ecr.AvailabilityStatus = tt

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err.Error() != want {
			t.Errorf("want '%s', got '%s'", want, err)
		}
	}
}

// TestValidateEndpointEditRequest tests that when a proper EndpointEditRequest is given, the validator doesn't
// complain.
func TestValidateEndpointEditRequest(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
	if err != nil {
		t.Errorf(`unexpected error when validating a valid "edit endpoint" payload: %s`, err)
	}
}

// TestEditDefaultEndpointAlreadyExists tests if an error is returned when the provided endpoint is marked as default
// when the also provided source already has a default one. Tested on an "edit endpoint" payload.
func TestEditDefaultEndpointAlreadyExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()
	tmp := true
	editRequest.Default = &tmp

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	want := "a default endpoint already exists for the provided source"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error when marking an endpoint as the default one when there's another default endpoint for the source. Want "%s" error, got "%s"`, want, got)
	}
}

// TestEditNonUniqueRole tests if an error is returned when a role already exists for a source. Tested on an "edit
// endpoint" payload.
func TestEditNonUniqueRole(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	role := "myRole"
	newEndpoint := fixtures.TestEndpointData[0]
	newEndpoint.ID = 0 // set it as zero to avoid hitting
	newEndpoint.Role = &role

	err := endpointDao.Create(&newEndpoint)
	if err != nil {
		t.Errorf("unexpected error when inserting a new endpoint for the test: %s", err)
	}

	editRequest, sourceId := setUpEndpointEditRequest()
	editRequest.Role = &role

	err = ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	want := "the role already exists for the given source"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error when modifying an endpoint's role to an already existing one. Want "%s", got "%s"`, want, got)
	}
}

// TestEditSchemeGetsDefaulted tests if the scheme gets properly defaulted when an invalid or missing scheme is
// provided. Tested on an "edit endpoint" payload.
func TestEditSchemeGetsDefaulted(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	editRequest.Scheme = nil

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	if err != nil {
		t.Errorf("unexpected error when validating a nil scheme for an endpoint edit: %s", err)
	}

	if *editRequest.Scheme != defaultScheme {
		t.Errorf(`unexpected scheme returned when editing an endpoint and not providing an scheme. Want "%s", got "%s"`, defaultScheme, *editRequest.Scheme)
	}

	invalidScheme := "/invalid"
	editRequest.Scheme = &invalidScheme

	err = ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	if err != nil {
		t.Errorf("unexpected error when editing an endpoint and giving it an invalid scheme: %s", err)
	}

	if *editRequest.Scheme != defaultScheme {
		t.Errorf(`unexpected scheme set when editing an endpoint with an invalid scheme value. Want "%s", got "%s"`, defaultScheme, *editRequest.Scheme)
	}
}

// TestEditEmptyHost tests if no error is returned even when an empty host is given. Tested on an "edit endpoint"
// payload.
func TestEditEmptyHost(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()
	tmp := ""
	editRequest.Host = &tmp

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
	if err != nil {
		t.Errorf("unexpected error when editing an endpoint and giving it an empty host: %s", err)
	}
}

// TestEditHostFqdnTooLong tests if an error is returned when a host which is too long is given. Tested on an "edit
// endpoint" payload.
func TestEditHostFqdnTooLong(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	// The "longHostname" variable holds a 256 char hostname
	tmp := `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee
		aaaaaa
	`
	editRequest.Host = &tmp

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	want := "the provided host is longer than 255 characters"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error when editing an endpoint and giving it a hostname that is too long. Want "%s" got "%s"`, want, got)
	}
}

// TestEditValidHosts tests if the validation succeeds when valid hosts are given. Tested on an "edit endpoint"
// payload.
func TestEditValidHosts(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testValues := []string{
		"redhat.com",
		"elpmaxe.example.com",
		"exa.mp-le.com",
		"123456789.org",
		"123ab-ba321.org",
		"a.org",
		"5.com",
		"example.xn--whatever",
	}

	editRequest, sourceId := setUpEndpointEditRequest()

	for _, tt := range testValues {
		editRequest.Host = &tt

		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
		if err != nil {
			t.Errorf(`unexpected error when editing an endpoint and giving it valid hostnames: %s for %#v`, err, tt)
		}
	}
}

// TestEditInvalidHosts tests if an error is returned on invalid hosts. Tested on an "edit endpoint" payload.
func TestEditInvalidHosts(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	testValues := []string{
		"-example.com",
		"example-.com",
		"example.xn--",
		"example.--xn",
	}

	editRequest, sourceId := setUpEndpointEditRequest()

	want := "the provided host is not valid"
	for _, tt := range testValues {
		editRequest.Host = &tt

		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
		if err == nil {
			t.Errorf(`unexpected error when editing an endpoint and giving it invalid hostnames: want "%s", got no error for %#v`, want, tt)
		}
	}
}

// TestEditLabelNamesTooLong tests if an error is returned when the provided labels are longer than permitted. Tested
// on an "edit endpoint" payload.
func TestEditLabelNamesTooLong(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	// The first label is 64 characters long
	tmp := `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeffffffffffgggg.example.org`
	editRequest.Host = &tmp

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	want := fmt.Sprintf(`the label '%s' is greater than %d characters`, "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeffffffffffgggg", maxLabelLength)
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error when editing an endpoint and giving it a label with is too long. Want "%s", got "%s"`, want, err)
	}
}

// TestEditDefaultingPortWhenMissingOrLessThanZero test if the port is given a default value if it's missing or if it
// has been given a value equal or lower than zero. Tested on an "edit endpoint" payload.
func TestEditDefaultingPortWhenMissingOrLessThanZero(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	minusOne := -1
	zero := 0
	testValues := []struct {
		value *int
	}{
		{nil},
		{&minusOne},
		{&zero},
	}

	for _, tt := range testValues {
		editRequest.Port = tt.value
		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
		if err != nil {
			t.Errorf("unexpected error when editing an endpoint and giving it invalid port values: %s", err)
		}

		if *editRequest.Port != defaultPort {
			t.Errorf(`unexpected value when editing an endpoint and giving it invalid port avlues. Want "%d" as the resulting default port, got "%d"`, defaultPort, *editRequest.Port)
		}
	}
}

// TestEditPortLargeValue tests if an error is returned when a port that is greater than the maximum allowed port is
// given. Tested on an "edit endpoint" payload.
func TestEditPortLargeValue(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()
	largePort := 999999
	editRequest.Port = &largePort

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

	want := "invalid port number"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error when editing an endpoint and giving it a port which is too high. Want "%s", got "%s"`, want, err)
	}
}

// TestEditDefaultVerifySsl tests if the default value is set when no "VerifySSL" value is provided. Tested on an "edit
// endpoint" payload.
func TestEditDefaultVerifySsl(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()
	editRequest.VerifySsl = nil

	err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
	if err != nil {
		t.Errorf(`unexpected error when editing an endpoint and giving it a nil "verifySSL" value: %s`, err)
	}

	if *editRequest.VerifySsl != defaultVerifySsl {
		t.Errorf(`unexpected error when editing an endpoint and giving it a nil "verifySSL" value. Want "%t" as the default value, got "%t"`, defaultVerifySsl, *editRequest.VerifySsl)
	}
}

// TestEditEmptyCertificateAuthority tests if an error is returned when ssl verification is turned on but no certificate
// authority is given. Tested on an "edit endpoint" payload.
func TestEditEmptyCertificateAuthority(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	emptyString := ""
	testValues := []struct {
		value *string
	}{
		{nil},
		{&emptyString},
	}

	want := "the certificate authority cannot be empty"
	for _, tt := range testValues {
		editRequest.CertificateAuthority = tt.value

		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error when editing an endpoint and giving it an empty certificate authority. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestEditEndpointValidAvailabilityStatuses tests if no error is returned when valid availability statuses are given.
// Tested on an "edit endpoint" payload.
func TestEditEndpointValidAvailabilityStatuses(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	testValues := []string{model.Available, model.Unavailable}
	for _, tt := range testValues {
		editRequest.AvailabilityStatus = &tt

		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)
		if err != nil {
			t.Errorf("unexpected error when editing an endpoint and giving it valid availability status values: %s", err)
		}
	}
}

// TestEditEndpointInvalidAvailabilityStatuses tests if an error is returned when passing invalid availability statuses.
// Tested on an "edit endpoint" payload.
func TestEditEndpointInvalidAvailabilityStatuses(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	editRequest, sourceId := setUpEndpointEditRequest()

	invalidValues := []string{"hello", "world", "almost", "passes", "validation"}
	want := "invalid availability status"
	for _, tt := range invalidValues {
		editRequest.AvailabilityStatus = &tt

		err := ValidateEndpointEditRequest(endpointDao, sourceId, &editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error when editing an endpoint and giving it invalid availability status values. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestEditEndpointInvalidAvailabilityStatusPaused tests that an error is received when an invalid availability status
// is given when updating a paused endpoint.
func TestEditEndpointInvalidAvailabilityStatusPaused(t *testing.T) {
	testValues := []*string{
		util.StringRef(""),
		util.StringRef("availablel"),
		util.StringRef("inprogress"),
		util.StringRef("partial"),
		util.StringRef("unavalialbe"),
		util.StringRef(model.InProgress),
		util.StringRef(model.PartiallyAvailable),
	}

	want := `invalid availability status. Must be either "available" or "unavailable"`
	for _, tv := range testValues {

		editRequest := model.ResourceEditPausedRequest{
			AvailabilityStatus: tv,
		}

		endpoint := model.Endpoint{}
		err := endpoint.UpdateFromRequestPaused(&editRequest)

		got := err.Error()
		if want != got {
			t.Errorf(`unexpected error received when updating a paused endpoint with an invalid availability status. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestEditEndpointValidAvailabilityStatusPaused tests that no error is returned when valid availability statuses are
// provided when updating a paused endpoint.
func TestEditEndpointValidAvailabilityStatusPaused(t *testing.T) {
	testValues := []*string{
		util.StringRef(model.Available),
		util.StringRef(model.Unavailable),
	}

	for _, tv := range testValues {
		editRequest := model.ResourceEditPausedRequest{
			AvailabilityStatus: tv,
		}

		endpoint := model.Endpoint{}
		err := endpoint.UpdateFromRequestPaused(&editRequest)

		if err != nil {
			t.Errorf(`unexpected error when validating a valid availability status "%s" for a paused endpoint edit: %s`, *tv, err)
		}
	}
}
