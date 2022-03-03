package statuslistener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/types"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// setUpKafkaHeaders sets up the required Kafka headers that the status listener will be looking for.
func setUpKafkaHeaders() []kafkaGo.Header {
	eventTypeHeader := kafkaGo.Header{
		Key:   "event_type",
		Value: []byte("availability_status"),
	}

	// {"identity":{"account_number":"12345","user": {"is_org_admin":true}}, "internal": {"org_id": "000001"}}
	xRhIdentityHeader := kafkaGo.Header{
		Key:   "x-rh-identity",
		Value: []byte("eyJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IjEyMzQ1IiwidXNlciI6IHsiaXNfb3JnX2FkbWluIjp0cnVlfX0sICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fQo="),
	}

	xRhSourcesAccountNumberHeader := kafkaGo.Header{
		Key:   "x-rh-sources-account-number",
		Value: []byte("12345"),
	}

	return []kafkaGo.Header{
		eventTypeHeader,
		xRhIdentityHeader,
		xRhSourcesAccountNumberHeader,
	}
}

// MockFormatter is just a formatter so that the logging works in the tests.
type MockFormatter struct {
	Hostname              string
	AppName               string
	InjectedToOtherLogger bool
}

func (m MockFormatter) Format(_ *logrus.Entry) ([]byte, error) {
	return []byte{}, nil
}

// setUpTests sets up the mocked Vault DAO and the logger so that the functions under test don't panic with a
// dereference error.
func setUpTests() {
	dao.Vault = &mocks.MockVault{}

	logging.Log = &logrus.Logger{
		Out:          os.Stdout,
		Level:        logrus.DebugLevel,
		Formatter:    MockFormatter{},
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: false,
	}
}

// MockEventStreamSender is a mock for the "RaiseEvent" function, which gets called every time the status listener
// processes an event.
type MockEventStreamSender struct{}

// getStatusMessageAndTestUtility is a function which allows returning the status message under test and the testing
// utilities, so that any functions that need them can use them. This is useful when the "RaiseEvent" function calls
// the "testRaiseEventData" function, since that one needs the status message used in the test.
var getStatusMessageAndTestUtility func() (types.StatusMessage, *testing.T)

// testRaiseEventWasCalled is a variable which will tell us if the "RaiseEvent" was called or not.
var testRaiseEventWasCalled bool

func (streamProducerSender *MockEventStreamSender) RaiseEvent(eventType string, payload []byte, _ []kafka.Header) error {
	testRaiseEventWasCalled = true

	// Get the status message and the test suite from the running test. This function must be set on each test for this
	// to work!
	statusMessage, t := getStatusMessageAndTestUtility()

	var expectedData []byte
	if eventType == "Records.update" {
		bulkMessage, err := getResourceBulkMessage(statusMessage)
		if err != nil {
			t.Errorf(`unexpected error when building the bulk message: %s`, err)
		}

		expectedData, err = json.Marshal(bulkMessage)
		if err != nil {
			t.Errorf(`unexpected error when marshalling the bulk message: %s`, err)
		}
	} else {
		event, err := getResourceAsEvent(statusMessage)
		if err != nil {
			t.Errorf(`unexpected error when pulling the resource: %s`, err)
		}

		expectedData, err = json.Marshal(event)
		if err != nil {
			t.Errorf(`unexpected error when marshalling the event: %s`, err)
		}
	}

	if !bytes.Equal(expectedData, payload) {
		errMsg := "error in raising event of type %s with resource type %s.\n" +
			"JSON payloads are not same:\n\n" +
			"Expected: %s \n" +
			"Obtained: %s \n"

		t.Errorf(errMsg, eventType, statusMessage.ResourceType, expectedData, payload)
		return fmt.Errorf(errMsg, eventType, statusMessage.ResourceType, expectedData, payload)
	}

	return nil
}

type ExpectedData struct {
	BulkMessage  []byte
	ResourceJSON []byte
}

type TestData struct {
	types.StatusMessage
	m.AvailabilityStatus
	ExpectedData

	MessageHeaders []kafkaGo.Header

	RaiseEventCalled bool
}

// TestConsumeStatusMessageSource tests whether a proper event is generated when a source's availability status gets
// updated.
func TestConsumeStatusMessageSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	setUpTests()
	kafkaHeaders := setUpKafkaHeaders()

	statusMessageSource := types.StatusMessage{
		ResourceType: "Source",
		ResourceID:   "1",
		Status:       m.Available,
	}

	sourceTestData := TestData{
		StatusMessage:    statusMessageSource,
		MessageHeaders:   kafkaHeaders,
		RaiseEventCalled: true,
	}

	// testRaiseEventWasCalled must be set to false every time to avoid issues with tests which require different
	// values.
	testRaiseEventWasCalled = false
	sender := MockEventStreamSender{}
	esp := &events.EventStreamProducer{Sender: &sender}
	avs := AvailabilityStatusListener{EventStreamProducer: esp}
	message, _ := json.Marshal(sourceTestData)

	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageSource, t
	}

	avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: sourceTestData.MessageHeaders})

	if testRaiseEventWasCalled != sourceTestData.RaiseEventCalled {
		t.Errorf(`Was raise event called? Want: "%t", got "%t"`, sourceTestData.RaiseEventCalled, testRaiseEventWasCalled)
	}
}

// TestConsumeStatusMessageApplication tests whether a proper event is generated when a source's availability status
// gets updated.
func TestConsumeStatusMessageApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	setUpTests()
	kafkaHeaders := setUpKafkaHeaders()

	statusMessageApplication := types.StatusMessage{
		ResourceType: "Application",
		ResourceID:   "1",
		Status:       m.Available,
	}

	applicationTestData := TestData{
		StatusMessage:    statusMessageApplication,
		MessageHeaders:   kafkaHeaders,
		RaiseEventCalled: true,
	}

	// testRaiseEventWasCalled must be set to false every time to avoid issues with tests which require different
	// values.
	testRaiseEventWasCalled = false
	sender := MockEventStreamSender{}
	esp := &events.EventStreamProducer{Sender: &sender}
	avs := AvailabilityStatusListener{EventStreamProducer: esp}
	message, _ := json.Marshal(applicationTestData)

	// Set the status message and the testing utilities which will be pulled in the "RaiseEvent" function of this test
	// file.
	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageApplication, t
	}

	// This will end up calling the "RaiseEvent" function we've got in this test file.
	avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: applicationTestData.MessageHeaders})

	if testRaiseEventWasCalled != applicationTestData.RaiseEventCalled {
		t.Errorf(`Was raise event called? Want: "%t", got "%t"`, applicationTestData.RaiseEventCalled, testRaiseEventWasCalled)
	}
}

// TestConsumeStatusMessageEndpoint tests whether a proper event is generated when a source's availability status gets
// updated.
func TestConsumeStatusMessageEndpoint(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	setUpTests()
	kafkaHeaders := setUpKafkaHeaders()

	statusMessageEndpoint := types.StatusMessage{
		ResourceType: "Endpoint",
		ResourceID:   "1",
		Status:       m.Available,
	}

	endpointTestData := TestData{
		StatusMessage:    statusMessageEndpoint,
		MessageHeaders:   kafkaHeaders,
		RaiseEventCalled: true,
	}

	// testRaiseEventWasCalled must be set to false every time to avoid issues with tests which require different
	// values.
	testRaiseEventWasCalled = false

	sender := MockEventStreamSender{}
	esp := &events.EventStreamProducer{Sender: &sender}
	avs := AvailabilityStatusListener{EventStreamProducer: esp}
	message, _ := json.Marshal(endpointTestData)

	// Set the status message and the testing utilities which will be pulled in the "RaiseEvent" function of this test
	// file.
	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageEndpoint, t
	}

	// This will end up calling the "RaiseEvent" function we've got in this test file.
	avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: endpointTestData.MessageHeaders})

	if testRaiseEventWasCalled != endpointTestData.RaiseEventCalled {
		t.Errorf(`Was raise event called? Want: "%t", got "%t"`, endpointTestData.RaiseEventCalled, testRaiseEventWasCalled)
	}
}

// TestConsumeStatusMessageEndpointNotFound tests that when a non-existing endpoint gets a availability status update,
// no events are raised.
func TestConsumeStatusMessageEndpointNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	setUpTests()
	kafkaHeaders := setUpKafkaHeaders()

	statusMessageEndpoint := types.StatusMessage{
		ResourceType: "Endpoint",
		ResourceID:   "99",
		Status:       m.Available,
	}

	endpointTestDataNotFound := TestData{
		StatusMessage:    statusMessageEndpoint,
		MessageHeaders:   kafkaHeaders,
		RaiseEventCalled: false,
	}

	// testRaiseEventWasCalled must be set to false every time to avoid issues with tests which require different
	// values.
	testRaiseEventWasCalled = false

	sender := MockEventStreamSender{}
	esp := &events.EventStreamProducer{Sender: &sender}
	avs := AvailabilityStatusListener{EventStreamProducer: esp}
	message, _ := json.Marshal(endpointTestDataNotFound)

	// Set the status message and the testing utilities which will be pulled in the "RaiseEvent" function of this test
	// file.
	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageEndpoint, t
	}

	// This will end up calling the "RaiseEvent" function we've got in this test file.
	avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: endpointTestDataNotFound.MessageHeaders})

	if testRaiseEventWasCalled != endpointTestDataNotFound.RaiseEventCalled {
		t.Errorf(`Was RaiseEvent called? Want: "%t", got "%t"`, endpointTestDataNotFound.RaiseEventCalled, testRaiseEventWasCalled)
	}
}
