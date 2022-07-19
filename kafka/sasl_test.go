package kafka

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// These variables are used to create the Sasl configuration fixture. They are vars and not consts because a const
// can't be referenced.
var (
	fixtureSaslMechanism = scramSha512
	fixtureSaslUsername  = "username-fixture"
	fixtureSaslPassword  = "password-fixture"
)

func createKafkaSaslConfigurationFixture() *clowder.KafkaSASLConfig {
	return &clowder.KafkaSASLConfig{
		SaslMechanism: &fixtureSaslMechanism,
		Password:      &fixtureSaslPassword,
		Username:      &fixtureSaslUsername,
	}
}

// TestCreateDialerInvalidConfig tests that when a "nil" config is passed to the "CreateDialer" function, the function
// returns an error without panicking.
func TestCreateDialerInvalidConfig(t *testing.T) {
	_, err := CreateDialer(nil)

	want := "could not create a dialer for Kafka: the passed configuration is empty"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error received when creating a dialer. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateDialerEmptySasl tests that when empty Sasl settings are passed to the "CreateDialer" function, the
// function returns an error without panicking.
func TestCreateDialerEmptySasl(t *testing.T) {
	emptySaslConfig := clowder.BrokerConfig{}
	_, err := CreateDialer(&emptySaslConfig)

	want := "could not create a dialer for Kafka: the passed configuration is missing the Sasl settings"
	got := err

	if want != got.Error() {
		t.Errorf(`unexpected error received when creating a dialer. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateDialer tests that the dialer is properly set up when a proper configuration is passed to the function. We
// only test the results for a couple of options, since the other two are tested on the test functions below.
func TestCreateDialer(t *testing.T) {
	fakeCaCert := "test-ca-cert"

	config := &clowder.BrokerConfig{
		Cacert: &fakeCaCert,
		Sasl:   createKafkaSaslConfigurationFixture(),
	}

	dialer, err := CreateDialer(config)
	if err != nil {
		t.Errorf(`unexpected error when creating a dialer: %s`, err)
	}

	{
		want := true
		got := dialer.DualStack

		if want != got {
			t.Errorf(`unexpected "DualStack" value configured on the dialer. Want "%t", got "%t"`, want, got)
		}
	}

	{
		want := 10 * time.Second
		got := dialer.Timeout

		if want != got {
			t.Errorf(`unexpected "Timeout" value configured on the dialer. Want "%v", got "%v"`, want, got)
		}
	}
}

// TestCreateTlsConfigEmptyCaContents tests that when a nil or empty certificate is given to the function under test,
// the TLS configuration is only set up with the minimum TLS version.
func TestCreateTlsConfigEmptyCaContents(t *testing.T) {
	emptyString := ""
	testValues := []*string{nil, &emptyString}

	for _, tv := range testValues {
		tlsConfig := CreateTLSConfig(tv)

		{
			var want uint16 = tls.VersionTLS12
			got := tlsConfig.MinVersion

			if want != got {
				t.Errorf(`unexpected TLS's minimum version received. Want "%d", got "%d"`, want, got)
			}
		}

		{
			var want *x509.CertPool
			got := tlsConfig.RootCAs

			if want != got {
				t.Errorf(`unexpected root CAs found in the TLS configuration. Want none, got "%v"`, tlsConfig.RootCAs)
			}
		}
	}
}

// TestCreateTlsConfig tests that the TLS configuration is properly created when a valid configuration is given.
func TestCreateTlsConfig(t *testing.T) {
	// The certificate is a certificate from https://example.org
	caContents := `
-----BEGIN CERTIFICATE-----
MIIEvjCCA6agAwIBAgIQBtjZBNVYQ0b2ii+nVCJ+xDANBgkqhkiG9w0BAQsFADBh
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBD
QTAeFw0yMTA0MTQwMDAwMDBaFw0zMTA0MTMyMzU5NTlaME8xCzAJBgNVBAYTAlVT
MRUwEwYDVQQKEwxEaWdpQ2VydCBJbmMxKTAnBgNVBAMTIERpZ2lDZXJ0IFRMUyBS
U0EgU0hBMjU2IDIwMjAgQ0ExMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKC
AQEAwUuzZUdwvN1PWNvsnO3DZuUfMRNUrUpmRh8sCuxkB+Uu3Ny5CiDt3+PE0J6a
qXodgojlEVbbHp9YwlHnLDQNLtKS4VbL8Xlfs7uHyiUDe5pSQWYQYE9XE0nw6Ddn
g9/n00tnTCJRpt8OmRDtV1F0JuJ9x8piLhMbfyOIJVNvwTRYAIuE//i+p1hJInuW
raKImxW8oHzf6VGo1bDtN+I2tIJLYrVJmuzHZ9bjPvXj1hJeRPG/cUJ9WIQDgLGB
Afr5yjK7tI4nhyfFK3TUqNaX3sNk+crOU6JWvHgXjkkDKa77SU+kFbnO8lwZV21r
eacroicgE7XQPUDTITAHk+qZ9QIDAQABo4IBgjCCAX4wEgYDVR0TAQH/BAgwBgEB
/wIBADAdBgNVHQ4EFgQUt2ui6qiqhIx56rTaD5iyxZV2ufQwHwYDVR0jBBgwFoAU
A95QNVbRTLtm8KPiGxvDl7I90VUwDgYDVR0PAQH/BAQDAgGGMB0GA1UdJQQWMBQG
CCsGAQUFBwMBBggrBgEFBQcDAjB2BggrBgEFBQcBAQRqMGgwJAYIKwYBBQUHMAGG
GGh0dHA6Ly9vY3NwLmRpZ2ljZXJ0LmNvbTBABggrBgEFBQcwAoY0aHR0cDovL2Nh
Y2VydHMuZGlnaWNlcnQuY29tL0RpZ2lDZXJ0R2xvYmFsUm9vdENBLmNydDBCBgNV
HR8EOzA5MDegNaAzhjFodHRwOi8vY3JsMy5kaWdpY2VydC5jb20vRGlnaUNlcnRH
bG9iYWxSb290Q0EuY3JsMD0GA1UdIAQ2MDQwCwYJYIZIAYb9bAIBMAcGBWeBDAEB
MAgGBmeBDAECATAIBgZngQwBAgIwCAYGZ4EMAQIDMA0GCSqGSIb3DQEBCwUAA4IB
AQCAMs5eC91uWg0Kr+HWhMvAjvqFcO3aXbMM9yt1QP6FCvrzMXi3cEsaiVi6gL3z
ax3pfs8LulicWdSQ0/1s/dCYbbdxglvPbQtaCdB73sRD2Cqk3p5BJl+7j5nL3a7h
qG+fh/50tx8bIKuxT8b1Z11dmzzp/2n3YWzW2fP9NsarA4h20ksudYbj/NhVfSbC
EXffPgK2fPOre3qGNm+499iTcc+G33Mw+nur7SpZyEKEOxEXGlLzyQ4UfaJbcme6
ce1XR2bFuAJKZTRei9AqPCCcUZlM51Ke92sRKw2Sfh3oius2FkOH6ipjv3U/697E
A7sKPPcw7+uvTPyLNhBzPvOk
-----END CERTIFICATE-----
`

	tlsConfig := CreateTLSConfig(&caContents)

	{
		var want uint16 = tls.VersionTLS12
		got := tlsConfig.MinVersion

		if want != got {
			t.Errorf(`unexpected TLS's minimum version received. Want "%d", got "%d"`, want, got)
		}
	}

	subjects := tlsConfig.RootCAs.Subjects()
	{
		want := 1
		got := len(subjects)

		if want != got {
			t.Errorf(`unexpected number of certificates on the certificate pool. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := []byte("DigiCert")
		got := subjects[0]

		if !bytes.Contains(got, want) {
			t.Errorf(`unexpected certificate parsed. Want subject containing "%s", got "%s"`, want, got)
		}
	}
}

// TestCreateSaslMechanismEmptyConfig tests that when a "nil" configuration is passed to the function, an error is
// returned without panicking.
func TestCreateSaslMechanismEmptyConfig(t *testing.T) {
	_, err := CreateSaslMechanism(nil)

	want := "could not create a Sasl mechanism for Kafka: the Sasl configuration settings are empty"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error received when creating a Sasal mechanism. Want "%s", got "%s"`, want, got)
	}
}

// TestCreatSaslMechanismEmptySaslMechanism tests that when a "nil" or empty "SaslMechanism" is passed via the config,
// an error is returned without panicking.
func TestCreatSaslMechanismEmptySaslMechanism(t *testing.T) {
	emptyString := ""
	testValues := []*string{nil, &emptyString}

	saslConfig := createKafkaSaslConfigurationFixture()

	want := "could not create a Sasl mechanism for Kafka: the Sasl mechanism is empty"
	for _, tv := range testValues {
		saslConfig.SaslMechanism = tv

		_, err := CreateSaslMechanism(saslConfig)

		got := err.Error()

		if want != got {
			t.Errorf(`unexpected error received when creating a Sasal mechanism. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestCreateSaslMechanismNilUsername tests that when a "nil" username is passed via the config, an error is returned
// without panicking.
func TestCreateSaslMechanismNilUsername(t *testing.T) {
	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.Username = nil

	_, err := CreateSaslMechanism(saslConfig)

	want := "could not create a Sasl mechanism for Kafka: the Sasl username is nil"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error received when creating a Sasal mechanism. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslMechanismNilPassword tests that when a "nil" password is passed via the config, an error is returned
// without panicking.
func TestCreateSaslMechanismNilPassword(t *testing.T) {
	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.Password = nil

	_, err := CreateSaslMechanism(saslConfig)

	want := "could not create a Sasl mechanism for Kafka: the Sasl password is nil"
	got := err.Error()

	if want != got {
		t.Errorf(`unexpected error received when creating a Sasal mechanism. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslPlainMechanism tests that when a "plain" Sasl mechanism is provided, the corresponding kafka-go
// compatible mechanism is returned.
func TestCreateSaslPlainMechanism(t *testing.T) {
	plainMech := plain.Mechanism{}
	plainMechName := plainMech.Name()

	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.SaslMechanism = &plainMechName

	mechanism, err := CreateSaslMechanism(saslConfig)
	if err != nil {
		t.Errorf(`unexpected error when creating a "plain" Sasl mechanism: %s`, err)
	}

	want := plainMech.Name()
	got := mechanism.Name()
	if want != got {
		t.Errorf(`unexpected Sasl mechanism created. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslSha512Mechanism tests that when a "scram-sha-256" Sasl mechanism is provided, the corresponding
// kafka-go compatible mechanism is returned.
func TestCreateSaslSha512Mechanism(t *testing.T) {
	scramSha256 := scram.SHA256.Name()

	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.SaslMechanism = &scramSha256

	mechanism, err := CreateSaslMechanism(saslConfig)
	if err != nil {
		t.Errorf(`unexpected error when creating a "scram-sha-256" Sasl mechanism: %s`, err)
	}

	want := scram.SHA256.Name()
	got := mechanism.Name()
	if want != got {
		t.Errorf(`unexpected Sasl mechanism created. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslSha512Mechanism tests that when a "scram-sha-512" Sasl mechanism is provided, the corresponding
// kafka-go compatible mechanism is returned.
func TestCreateSaslScramSha512Mechanism(t *testing.T) {
	scramSha512 := scram.SHA512.Name()

	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.SaslMechanism = &scramSha512

	mechanism, err := CreateSaslMechanism(saslConfig)
	if err != nil {
		t.Errorf(`unexpected error when creating a "scram-sha-512" Sasl mechanism: %s`, err)
	}

	want := scram.SHA512.Name()
	got := mechanism.Name()
	if want != got {
		t.Errorf(`unexpected Sasl mechanism created. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslInvalidMechanism tests that when a not recognized Sasl mechanism is provided, an error is returned.
func TestCreateSaslInvalidMechanism(t *testing.T) {
	whatever := "whatever"

	saslConfig := createKafkaSaslConfigurationFixture()
	saslConfig.SaslMechanism = &whatever

	_, err := CreateSaslMechanism(saslConfig)

	want := `unable to configure Sasl mechanism "whatever" for Kafka`
	got := err.Error()
	if want != got {
		t.Errorf(`unexpected error when providing an invalid Sasl mechanism. Want "%s", got "%s"`, want, got)
	}
}
