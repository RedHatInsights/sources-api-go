package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// The following constant variables are used on the Sasl mechanism creation.
const (
	saslPlain   = "plain"
	scramSha256 = "scram-sha-256"
	scramSha512 = "scram-sha-512"
)

var (
	// package level sharable kafka variables, safe to use concurrently.
	Dialer    *kafka.Dialer
	Transport *kafka.Transport

	TlsConfig     *tls.Config
	SaslMechanism sasl.Mechanism
)

// CreateDialer returns a Kafka dialer for the Kafka Go library, which includes the TLS configuration and the Sasl
// mechanism to connect to the managed Kafka.
func CreateDialer(config *clowder.BrokerConfig) (*kafka.Dialer, error) {
	if Dialer != nil {
		return Dialer, nil
	}

	if config == nil {
		return nil, errors.New(`could not create a dialer for Kafka: the passed configuration is empty`)
	}

	if config.Sasl == nil {
		return nil, errors.New(`could not create a dialer for Kafka: the passed configuration is missing the Sasl settings`)
	}

	tlsConfig := CreateTLSConfig(config.Cacert)

	mechanism, err := CreateSaslMechanism(config.Sasl)
	if err != nil {
		return nil, fmt.Errorf(`unable to create the Sasl mechanism for the dialer: %w`, err)
	}

	Dialer = &kafka.Dialer{
		DualStack:     true,
		SASLMechanism: mechanism,
		Timeout:       10 * time.Second,
		TLS:           tlsConfig,
	}

	return Dialer, nil
}

// CreateTLSConfig returns a TLS configuration. The minimum TLS version is set to 1.2 and if the "caContents" are not
// empty the provided certificate is included as "trusted" for the TLS configuration.
func CreateTLSConfig(caContents *string) *tls.Config {
	if TlsConfig != nil {
		return TlsConfig
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// The managed Kafka instance may have a self-signed certificate, and in those cases we must be able to support the
	// connection too.
	if caContents != nil && *caContents != "" {
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(*caContents))

		tlsConfig.RootCAs = certPool
	}

	TlsConfig = tlsConfig

	return TlsConfig
}

// CreateSaslMechanism returns a Sasl mechanism that Kafka Go requires for setting up the connection. Currently, we
// support plain, scram-sha-256 and scram-sha-512 mechanisms.
func CreateSaslMechanism(saslConfig *clowder.KafkaSASLConfig) (sasl.Mechanism, error) {
	if SaslMechanism != nil {
		return SaslMechanism, nil
	}

	if saslConfig == nil {
		return nil, errors.New("could not create a Sasl mechanism for Kafka: the Sasl configuration settings are empty")
	}

	if saslConfig.SaslMechanism == nil || *saslConfig.SaslMechanism == "" {
		return nil, errors.New("could not create a Sasl mechanism for Kafka: the Sasl mechanism is empty")
	}

	if saslConfig.Username == nil {
		return nil, errors.New("could not create a Sasl mechanism for Kafka: the Sasl username is nil")
	}

	if saslConfig.Password == nil {
		return nil, errors.New("could not create a Sasl mechanism for Kafka: the Sasl password is nil")
	}

	var algorithm scram.Algorithm

	switch strings.ToLower(*saslConfig.SaslMechanism) {
	case saslPlain:
		return plain.Mechanism{Username: *saslConfig.Username, Password: *saslConfig.Password}, nil
	case scramSha256:
		algorithm = scram.SHA256
	case scramSha512:
		algorithm = scram.SHA512
	default:
		return nil, fmt.Errorf(`unable to configure Sasl mechanism "%s" for Kafka`, *saslConfig.SaslMechanism)
	}

	mechanism, err := scram.Mechanism(algorithm, *saslConfig.Username, *saslConfig.Password)
	if err != nil {
		return nil, fmt.Errorf(`unable to generate "%s" mechanism with the "%v" algorithm: %s`, *saslConfig.SaslMechanism, algorithm, err)
	}

	SaslMechanism = mechanism

	return SaslMechanism, nil
}

// CreateTransport returns a kafka transport that is memoized since it can be used concurrently
func CreateTransport(sasl sasl.Mechanism, tls *tls.Config) *kafka.Transport {
	if Transport != nil {
		return Transport
	}

	Transport = &kafka.Transport{SASL: sasl, TLS: tls}

	return Transport
}
