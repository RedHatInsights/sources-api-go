package kafka

import (
	"testing"

	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// TestCreateSaslPlainMechanism tests that when a "plain" Sasl mechanism is provided, the corresponding kafka-go
// compatible mechanism is returned.
func TestCreateSaslPlainMechanism(t *testing.T) {
	mechanism, err := CreateSaslMechanism("plain", "username", "password")
	if err != nil {
		t.Errorf(`unexpected error when creating a "plain" Sasl mechanism: %s`, err)
	}

	var plainMech = plain.Mechanism{}
	want := plainMech.Name()

	got := mechanism.Name()
	if want != got {
		t.Errorf(`unexpected Sasl mechanism created. Want "%s", got "%s"`, want, got)
	}
}

// TestCreateSaslSha512Mechanism tests that when a "scram-sha-256" Sasl mechanism is provided, the corresponding
// kafka-go compatible mechanism is returned.
func TestCreateSaslSha512Mechanism(t *testing.T) {
	mechanism, err := CreateSaslMechanism("scram-sha-256", "username", "password")
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
	mechanism, err := CreateSaslMechanism("scram-sha-512", "username", "password")
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
	_, err := CreateSaslMechanism("whatever", "username", "password")

	want := `unable to configure Sasl mechanism "whatever" for Kafka`
	got := err.Error()
	if want != got {
		t.Errorf(`unexpected error when providing an invalid Sasl mechanism. Want "%s", got "%s"`, want, got)
	}
}
