package service

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/model"
)

func setUpEndpointCreateRequest() model.EndpointCreateRequest {
	receptorNode := "receptorNode"
	scheme := "https"
	port := int64(443)
	verifySsl := true
	certificateAuthority := "letsEncrypt"
	sourceId := strconv.FormatInt(testutils.TestSourceData[0].ID, 10)

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

// TestValidateEndpointCreateRequest tests that when a proper EndpointCreateRequest is given, the validator doesn't
// complain.
func TestValidateEndpointCreateRequest(t *testing.T) {
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

	role := "myRole"
	newEndpoint := testutils.TestEndpointData[0]
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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

	ecr := setUpEndpointCreateRequest()
	ecr.Host = ""

	err := ValidateEndpointCreateRequest(endpointDao, &ecr)
	if err != nil {
		t.Errorf("want no error, got '%s'", err)
	}
}

// TestHostFqdnTooLong tests if an error is returned when a host which is too long is given.
func TestHostFqdnTooLong(t *testing.T) {
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

	ecr := setUpEndpointCreateRequest()

	minusOne := int64(-1)
	zero := int64(0)
	testValues := []struct {
		value *int64
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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

	ecr := setUpEndpointCreateRequest()
	largePort := int64(999999)
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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

	ecr := setUpEndpointCreateRequest()

	testValues := []string{"", model.Available, model.Unavailable}
	for _, tt := range testValues {
		ecr.AvailabilityStatus = tt

		err := ValidateEndpointCreateRequest(endpointDao, &ecr)
		if err != nil {
			t.Errorf("want no error, got '%s'", err)
		}
	}
}

// TestInvalidAvailabilityStatuses tests if an error is returned when passing invalid availability statuses.
func TestInvalidAvailabilityStatuses(t *testing.T) {
	if !runningIntegration {
		t.Skip("skipping integration tests...")
	}

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
