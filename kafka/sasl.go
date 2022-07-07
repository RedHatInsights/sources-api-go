package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

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

// CreateDialer returns a Kafka dialer for the Kafka Go library, which includes the TLS configuration and the Sasl
// mechanism to connect to the managed Kafka.
func CreateDialer(kafkaSaslCertPath string, saslMechanism string, username string, password string) (*kafka.Dialer, error) {
	tlsConfig, err := CreateTLSConfig(kafkaSaslCertPath)
	if err != nil {
		return nil, fmt.Errorf(`unable to create the TLS config for the dialer: %w`, err)
	}

	mechanism, err := CreateSaslMechanism(saslMechanism, username, password)
	if err != nil {
		return nil, fmt.Errorf(`unable to create the Sasl mechanism for the dialer: %w`, err)
	}

	return &kafka.Dialer{
		DualStack:     true,
		SASLMechanism: mechanism,
		Timeout:       10 * time.Second,
		TLS:           tlsConfig,
	}, nil
}

// CreateTLSConfig returns a TLS configuration including the managed Kafka's CA and a minimum required TLS version
// configuration option so that the Kafka Go client knows how to properly connect to our managed Kafka.
func CreateTLSConfig(kafkaSaslCertPath string) (*tls.Config, error) {
	cert, err := ioutil.ReadFile(kafkaSaslCertPath)
	if err != nil {
		return nil, fmt.Errorf(`could not read the Kafka certificate file from "%s": %s`, kafkaSaslCertPath, err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	return &tls.Config{
		MinVersion: tls.VersionTLS12, // required to avoid unexpected EOF errors
		RootCAs:    certPool,
	}, nil
}

// CreateSaslMechanism returns a Sasl mechanism that Kafka Go requires for setting up the connection. Currently, we
// support plain, scram-sha-256 and scram-sha-512 mechanisms.
func CreateSaslMechanism(saslMechanism string, username string, password string) (sasl.Mechanism, error) {
	var algorithm scram.Algorithm
	switch strings.ToLower(saslMechanism) {
	case saslPlain:
		return plain.Mechanism{Username: username, Password: password}, nil
	case scramSha256:
		algorithm = scram.SHA256
	case scramSha512:
		algorithm = scram.SHA512
	default:
		return nil, fmt.Errorf(`unable to configure Sasl mechanism "%s" for Kafka`, saslMechanism)
	}

	mechanism, err := scram.Mechanism(algorithm, username, password)
	if err != nil {
		return nil, fmt.Errorf(`unable to generate "%s" mechanism with the "%v" algorithm: %s`, saslMechanism, algorithm, err)
	}

	return mechanism, nil
}
