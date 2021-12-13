package kafka

import (
	"testing"

	"github.com/segmentio/kafka-go/protocol"
)

func TestGetHeader(t *testing.T) {
	m := Message{Headers: []protocol.Header{
		{Key: "good", Value: []byte("information")}},
	}

	if m.GetHeader("good") != "information" {
		t.Error("Message#GetHeader(string) failed to fetch the right header")
	}

	if m.GetHeader("bad") != "" {
		t.Error("Message#GetHeader(string) found a header where none should exist")
	}
}

func TestAddValueAsJSON(t *testing.T) {
	m := Message{}

	val := map[string]interface{}{"thing": true}

	err := m.AddValueAsJSON(val)
	if err != nil {
		t.Errorf("ran into error marshaling json: %v", err)
	}

	if string(m.Value) != `{"thing":true}` {
		t.Errorf("Message#AddValueAsJSON did not marshal object correctly, wanted %q got %q", `{"thing":true}`, string(m.Value))
	}
}

func TestIsEmpty(t *testing.T) {
	m := Message{}

	if !m.isEmpty() {
		t.Error("Message#IsEmpty showed not empty even though it is an empty struct")
	}

	m.AddHeaders([]Header{
		{Key: "foo", Value: []byte("bar")},
	})

	if m.isEmpty() {
		t.Error("Message#IsEmpty showed empty even though it is not an empty struct")
	}
}
