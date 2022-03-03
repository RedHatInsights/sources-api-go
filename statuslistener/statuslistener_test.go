package statuslistener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
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

// getSourceMap creates the expected JSON structure that the status listener will produce when receiving a status
// update. The function fetches the given source, its related applications, authentications and endpoints. Then, it
// creates a base map which has all that information, ready to be edited at will.
func getSourceMap(sourceId int64) (map[string]interface{}, error) {
	// resultingMap will mimic the structure of how the bulk message needs to look like. The idea is to fill it with
	// the required data that will be pulled from the database.
	var resultingMap = make(map[string]interface{})

	// Initialize the structure that we expect the status listener to build.
	emptyArray := []interface{}{}

	resultingMap["application_authentications"] = emptyArray
	resultingMap["applications"] = emptyArray
	resultingMap["authentications"] = emptyArray
	resultingMap["endpoints"] = emptyArray

	empty := struct{}{}
	resultingMap["source"] = empty
	resultingMap["updated"] = empty

	// rawData will hold all the data that needs to be set in the resultingMap. As all the models return an
	// "interface{}", it needs to be of that generic type.
	var rawData []interface{}

	// dbSource is the main source that connects everything in the bulk message.
	var dbSource m.Source

	// Pull the application authentications for the given source.
	var appAuths = make([]m.ApplicationAuthentication, 0)
	err := dao.DB.
		Preload(`Tenant`).
		Joins(`INNER JOIN applications ON "application_authentications"."application_id" = applications.id`).
		Where(`applications.source_id = ?`, sourceId).
		Find(&appAuths).
		Error

	if err != nil {
		return nil, err
	}

	// Append the data to the "rawData" and set it in the resultingMap.
	for _, appAuth := range appAuths {
		rawData = append(rawData, appAuth.ToEvent())
	}
	resultingMap["application_authentications"] = rawData

	err = dao.DB.
		Preload(`Applications`).
		Preload(`Applications.Tenant`).
		Preload(`Endpoints`).
		Preload(`Endpoints.Tenant`).
		Preload(`Tenant`).
		Where(`id = ?`, sourceId).
		Find(&dbSource).
		Error

	if err != nil {
		return nil, err
	}

	// Reset the rawData, so we don't include the previous data and append the data to the "rawData" and set it in the
	// resultingMap.
	rawData = []interface{}{}
	for _, app := range dbSource.Applications {
		rawData = append(rawData, app.ToEvent())
	}
	resultingMap["applications"] = rawData

	// Pull the authentications for that given source.
	authentication := &m.Authentication{ResourceID: sourceId, ResourceType: "Source"}
	authDao := dao.AuthenticationDaoImpl{TenantID: &fixtures.TestTenantData[0].Id}

	authentications, err := authDao.AuthenticationsByResource(authentication)
	if err != nil {
		return nil, err
	}

	// Reset the rawData, so we don't include the previous data and append the data to the "rawData" and set it in the
	// resultingMap.
	rawData = []interface{}{}
	for _, auth := range authentications {
		// The tenant must be set the same way it is done in dao/common.go#BulkMessageFromSource
		auth.Tenant = fixtures.TestTenantData[0]
		rawData = append(rawData, auth.ToEvent())
	}
	resultingMap["authentications"] = rawData

	// Reset the rawData, so we don't include the previous data and append the data to the "rawData" and set it in the
	// resultingMap.
	rawData = []interface{}{}
	for _, endpoint := range dbSource.Endpoints {
		rawData = append(rawData, endpoint.ToEvent())
	}
	resultingMap["endpoints"] = rawData

	// Include the source in the map as well.
	resultingMap["source"] = dbSource.ToEvent()

	return resultingMap, nil
}

// getSourceBulkMessage generates the expected bulk message that the status listener should generate. It returns a map
// with the ideal structure, ready to be marshalled.
func getSourceBulkMessage(sourceId int64) (map[string]interface{}, error) {
	resultingMap, err := getSourceMap(sourceId)
	if err != nil {
		return nil, err
	}

	// We expect the following fields to be updated from the Source.
	updatedMetadata := map[string]interface{}{
		"Source": map[string]interface{}{
			"1": []string{"availability_status", "last_available_at", "last_checked_at"},
		},
	}
	resultingMap["updated"] = updatedMetadata

	return resultingMap, nil
}

// getApplicationBulkMessage returns a map representing what the status listener is expected to generate.
func getApplicationBulkMessage(sourceId int64) (map[string]interface{}, error) {
	// Ideally the function should hit the database to fetch all the related data for the application, but since all
	// the fixtures belong to the same source, we simply fetch everything with the helper "getSourceMap" function and
	// remove the bits we're not interested in. If the fixtures change in the future, we will have to refactor this.
	resultingMap, err := getSourceMap(sourceId)
	if err != nil {
		return nil, err
	}

	// We expect the following fields to be updated from the Source.
	updatedMetadata := map[string]interface{}{
		"Application": map[string]interface{}{
			"1": []string{"availability_status", "availability_status_error", "last_available_at", "last_checked_at"},
		},
	}
	resultingMap["updated"] = updatedMetadata

	return resultingMap, nil
}

// getEndpointBulkMessage generates the expected bulk message that the status listener should generate. It returns a
// map with the ideal structure, ready to be marshalled.
func getEndpointBulkMessage(sourceId int64) (map[string]interface{}, error) {
	// Ideally the function should hit the database to fetch all the related data for the endpoint, but since all
	// the fixtures belong to the same source, we simply fetch everything with the helper "getSourceMap" function and
	// remove the bits we're not interested in. If the fixtures change in the future, we will have to refactor this.
	resultingMap, err := getSourceMap(sourceId)
	if err != nil {
		return nil, err
	}

	// There will not be any application authentications related to the endpoint.
	rawData := []interface{}{}
	resultingMap["application_authentications"] = rawData
	resultingMap["authentications"] = rawData

	// We expect the following fields to be updated from the Source.
	updatedMetadata := map[string]interface{}{
		"Endpoint": map[string]interface{}{
			"1": []string{"availability_status", "availability_status_error", "last_available_at", "last_checked_at"},
		},
	}
	resultingMap["updated"] = updatedMetadata

	return resultingMap, nil

}

// getResourceAsEvent fetches the resource specified in the status message and returns it in the ".ToEvent" format.
func getResourceAsEvent(statusMessage types.StatusMessage) (interface{}, error) {
	var err error
	resourceId, err := strconv.ParseInt(statusMessage.ResourceID, 10, 64)
	if err != nil {
		return nil, err
	}

	var event interface{}
	switch statusMessage.ResourceType {
	case "Application":
		var application m.Application
		err := dao.DB.Preload("Tenant").Where(`id = ?`, resourceId).Find(&application).Error

		if err != nil {
			return nil, err
		}

		event = application.ToEvent()
	case "Endpoint":
		var endpoint m.Endpoint
		err := dao.DB.Preload("Tenant").Where(`id = ?`, resourceId).Find(&endpoint).Error

		if err != nil {
			return nil, err
		}

		event = endpoint.ToEvent()
	case "Source":
		sourceDao := dao.SourceDaoImpl{TenantID: &fixtures.TestTenantData[0].Id}

		source, err := sourceDao.GetByIdWithPreload(&resourceId, "Tenant")

		if err != nil {
			return nil, err
		}

		event = source.ToEvent()
	}

	return event, nil
}

// getResourceBulkMessage simply returns the ideal bulk message structure for the given resource. That ideal bulk
// message structure is what is expected for the status listener to generate.
func getResourceBulkMessage(statusMessage types.StatusMessage) (map[string]interface{}, error) {
	var err error
	resourceId, err := strconv.ParseInt(statusMessage.ResourceID, 10, 64)
	if err != nil {
		return nil, err
	}

	var bulkMessage map[string]interface{}
	switch statusMessage.ResourceType {
	case "Application":
		bulkMessage, err = getApplicationBulkMessage(resourceId)
		if err != nil {
			return nil, err
		}

	case "Endpoint":
		bulkMessage, err = getEndpointBulkMessage(resourceId)
		if err != nil {
			return nil, err
		}
	case "Source":
		bulkMessage, err = getSourceBulkMessage(resourceId)
		if err != nil {
			return nil, err
		}
	}

	return bulkMessage, nil
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
