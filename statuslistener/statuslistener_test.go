package statuslistener

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/types"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type MockFormatter struct {
	Hostname              string
	AppName               string
	InjectedToOtherLogger bool
}

func (m MockFormatter) Format(_ *logrus.Entry) ([]byte, error) {
	return []byte{}, nil
}

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

func LoadJSONContentFrom(resourceType string, resourceID string, prefix string) []byte {
	fileName := "./test_data/" + prefix + resourceType + "_" + resourceID + ".json"
	fileContent, err := os.ReadFile(fileName)

	if err != nil {
		panic(fmt.Errorf("unable to read file %s because of %s", fileName, err.Error()))
	}

	return fileContent
}

func BulkMessageFor(resourceType string, resourceID string) []byte {
	bulkMessage := LoadJSONContentFrom(resourceType, resourceID, "bulk_message_")
	return TransformDateFieldsInJSONForBulkMessage(resourceType, resourceID, bulkMessage)
}

type DateFields struct {
	CreatedAt       time.Time
	LastAvailableAt time.Time
	LastCheckedAt   time.Time
	UpdatedAt       time.Time
}

func UpdateDateFieldsTo(fieldsMap map[string]interface{}, dateFields DateFields) map[string]interface{} {
	if !dateFields.CreatedAt.IsZero() {
		fieldsMap["created_at"] = util.FormatTimeToString(dateFields.CreatedAt, util.RecordDateTimeFormat)
	}

	if !dateFields.LastAvailableAt.IsZero() {
		fieldsMap["last_available_at"] = util.FormatTimeToString(dateFields.LastAvailableAt, util.RecordDateTimeFormat)
	}

	if !dateFields.LastCheckedAt.IsZero() {
		fieldsMap["last_checked_at"] = util.FormatTimeToString(dateFields.LastCheckedAt, util.RecordDateTimeFormat)
	}

	if !dateFields.UpdatedAt.IsZero() {
		fieldsMap["updated_at"] = util.FormatTimeToString(dateFields.UpdatedAt, util.RecordDateTimeFormat)
	}

	return fieldsMap
}

func PopulateDateFieldsFrom(resource interface{}) DateFields {
	dateFields := DateFields{}

	switch typedResource := resource.(type) {
	case *m.Source:
		dateFields.CreatedAt = typedResource.CreatedAt
		dateFields.LastAvailableAt = typedResource.LastAvailableAt
		dateFields.LastCheckedAt = typedResource.LastCheckedAt
		dateFields.UpdatedAt = typedResource.UpdatedAt
	case *m.Application:
		dateFields.CreatedAt = typedResource.CreatedAt
		dateFields.LastAvailableAt = typedResource.LastAvailableAt
		dateFields.LastCheckedAt = typedResource.LastCheckedAt
		dateFields.UpdatedAt = typedResource.UpdatedAt
	case *m.Endpoint:
		dateFields.CreatedAt = typedResource.CreatedAt
		dateFields.LastAvailableAt = typedResource.LastAvailableAt
		dateFields.LastCheckedAt = typedResource.LastCheckedAt
		dateFields.UpdatedAt = typedResource.UpdatedAt
	case *m.ApplicationAuthentication:
		dateFields.CreatedAt = typedResource.CreatedAt
		dateFields.UpdatedAt = typedResource.UpdatedAt
	default:
		panic("unable to find type")
	}

	return dateFields
}

func FetchDataFor(resourceType string, resourceID string, forBulkMessage bool) (interface{}, map[string]interface{}) {
	id, err := util.InterfaceToInt64(resourceID)
	if err != nil {
		panic("conversion error + " + err.Error())
	}

	var src interface{}
	res := dao.DB

	bulkMessage := map[string]interface{}{}

	switch resourceType {
	case "Source":
		source := &m.Source{ID: id}
		if forBulkMessage {
			res = dao.DB.Preload("Applications").Preload("Endpoints")
		}
		res = res.Find(source)
		bulkMessage["applications"] = source.Applications
		bulkMessage["endpoints"] = source.Endpoints

		appIDs := make([]int64, len(source.Applications))
		for index, application := range source.Applications {
			appIDs[index] = application.ID
		}

		var aa []m.ApplicationAuthentication
		if len(appIDs) > 0 {
			dao.DB.Where("application_id IN ?", appIDs).Find(&aa)
			bulkMessage["application_authentications"] = aa
		}

		src = source
	case "Application":
		application := &m.Application{ID: id}
		if forBulkMessage {
			res = dao.DB.Preload("Source").Preload("Source.Applications").Preload("Source.Endpoints")
		}

		res = res.Find(application)
		bulkMessage["applications"] = application.Source.Applications
		bulkMessage["endpoints"] = application.Source.Endpoints

		authentication := &m.Authentication{ResourceID: application.ID,
			ResourceType:               "Application",
			ApplicationAuthentications: []m.ApplicationAuthentication{},
		}

		authDao := &dao.AuthenticationDaoImpl{TenantID: &application.TenantID}
		authenticationsByResource, err := authDao.AuthenticationsByResource(authentication)
		if err != nil {
			panic("error to fetch authentications: " + err.Error())
		}

		bulkMessage["authentications"] = authenticationsByResource

		if err != nil {
			panic("error in adding authentications: " + err.Error())
		}

		appIDs := make([]int64, len(application.Source.Applications))
		for index, app := range application.Source.Applications {
			appIDs[index] = app.ID
		}

		var aa []m.ApplicationAuthentication
		if len(appIDs) > 0 {
			dao.DB.Where("application_id IN ?", appIDs).Find(&aa)
			bulkMessage["application_authentications"] = aa
		}

		src = application
	case "Endpoint":
		endpoint := &m.Endpoint{ID: id}
		if forBulkMessage {
			res = dao.DB.Preload("Source").Preload("Source.Applications").Preload("Source.Endpoints")
		}
		res = res.Find(endpoint)
		bulkMessage["applications"] = endpoint.Source.Applications
		bulkMessage["endpoints"] = endpoint.Source.Endpoints

		authentication := &m.Authentication{ResourceID: endpoint.ID,
			ResourceType:               "Endpoint",
			ApplicationAuthentications: []m.ApplicationAuthentication{},
		}
		authDao := &dao.AuthenticationDaoImpl{TenantID: &endpoint.TenantID}
		authenticationsByResource, err := authDao.AuthenticationsByResource(authentication)
		if err != nil {
			return err, nil
		}

		if err != nil {
			panic("error in adding authentications: " + err.Error())
		}

		bulkMessage["authentications"] = authenticationsByResource

		src = endpoint
	default:
		panic("can't find resource type")
	}

	err = res.Error

	if err != nil {
		panic("Error fetch record " + resourceID)
	}

	return src, bulkMessage
}

func TransformDateFieldsInJSONForBulkMessage(resourceType string, resourceID string, content []byte) []byte {
	resource, bulkMessage := FetchDataFor(resourceType, resourceID, true)

	contentMap := make(map[string]interface{})
	err := json.Unmarshal(content, &contentMap)
	if err != nil {
		panic("unmarshalling error + " + err.Error())
	}

	dateFields := PopulateDateFieldsFrom(resource)
	contentMap["source"] = UpdateDateFieldsTo(contentMap["source"].(map[string]interface{}), dateFields)

	var applications []interface{}

	apps, success := bulkMessage["applications"].([]m.Application)
	if !success {
		panic("type assertion error: + " + err.Error())
	}

	for index, application := range apps {
		dateFields = PopulateDateFieldsFrom(&application)
		ap, success := contentMap["applications"].([]interface{})
		if !success {
			panic("type assertion error: + " + err.Error())
		}

		upd := UpdateDateFieldsTo(ap[index].(map[string]interface{}), dateFields)
		applications = append(applications, upd)
	}
	contentMap["applications"] = applications

	var endpoints []interface{}

	ends, success := bulkMessage["endpoints"].([]m.Endpoint)
	if !success {
		panic("type assertion error: + " + err.Error())
	}

	for index, endpoint := range ends {
		dateFields = PopulateDateFieldsFrom(&endpoint)
		ap, success := contentMap["endpoints"].([]interface{})
		if !success {
			panic("type assertion error: + " + err.Error())
		}

		upd := UpdateDateFieldsTo(ap[index].(map[string]interface{}), dateFields)
		endpoints = append(endpoints, upd)
	}

	contentMap["endpoints"] = endpoints

	var applicationAuthentications []interface{}

	if applicationAuthenticationsBulkMessage, ok := bulkMessage["application_authentications"].([]m.ApplicationAuthentication); ok {
		for index, applicationAuthentication := range applicationAuthenticationsBulkMessage {
			dateFields = PopulateDateFieldsFrom(&applicationAuthentication)
			if ap, ok := contentMap["application_authentications"].([]interface{}); ok {
				upd := UpdateDateFieldsTo(ap[index].(map[string]interface{}), dateFields)
				applicationAuthentications = append(applicationAuthentications, upd)
			} else {
				panic("type assertion error: + " + err.Error())
			}

		}
	}

	if applicationAuthentications != nil {
		contentMap["application_authentications"] = applicationAuthentications
	} else {
		contentMap["application_authentications"] = []m.ApplicationAuthentication{}
	}

	contentJSON, err := json.Marshal(contentMap)
	if err != nil {
		panic("marshalling error + " + err.Error())
	}

	return contentJSON
}

func TransformDateFieldsInJSONForResource(resourceType string, resourceID string, content []byte) []byte {
	resource, _ := FetchDataFor(resourceType, resourceID, false)

	contentMap := make(map[string]interface{})
	err := json.Unmarshal(content, &contentMap)
	if err != nil {
		panic("unmarshalling error + " + err.Error())
	}

	dateFields := PopulateDateFieldsFrom(resource)
	contentMap = UpdateDateFieldsTo(contentMap, dateFields)

	contentJSON, err := json.Marshal(contentMap)
	if err != nil {
		panic("marshalling error + " + err.Error())
	}

	return contentJSON
}

func ResourceJSONFor(resourceType string, resourceID string) []byte {
	content := LoadJSONContentFrom(resourceType, resourceID, "resource_")
	return TransformDateFieldsInJSONForResource(resourceType, resourceID, content)
}

func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

// MockEventStreamSender is a mock for the "RaiseEvent" function, which gets called every time the status listener
// processes an event.
type MockEventStreamSender struct{}

// getStatusMessageAndTestUtility is a function which allows returning the status message under test and the testing
// utilities, so that any functions that need them can use them. This is useful when the "RaiseEvent" function calls
// the "testRaiseEventData" function, since that one needs the status message used in the test.
var getStatusMessageAndTestUtility func() (types.StatusMessage, *testing.T)

// testRaiseEventData is a function which gets called from RaiseEvent, which helps us customize what we want to have
// checked on each test, in case different things need to be tested.
func testRaiseEventData(eventType string, payload []byte) error {
	statusMessage, t := getStatusMessageAndTestUtility()

	var isResult bool
	var expectedData []byte
	if eventType == "Records.update" {
		expectedData = BulkMessageFor(statusMessage.ResourceType, statusMessage.ResourceID)
	} else {
		expectedData = ResourceJSONFor(statusMessage.ResourceType, statusMessage.ResourceID)
	}

	isResult, err := JSONBytesEqual(payload, expectedData)
	if err != nil {
		t.Errorf("error with parsing JSON: %s", err.Error())
	}

	if isResult != true {
		errMsg := "error in raising event of type %s with resource type %s.\n" +
			"JSON payloads are not same:\n\n" +
			"Expected: %s \n" +
			"Obtained: %s \n"

		t.Errorf(errMsg, eventType, statusMessage.ResourceType, expectedData, payload)
		return fmt.Errorf(errMsg, eventType, statusMessage.ResourceType, expectedData, payload)
	}

	return nil
}

// testRaiseEventWasCalled is a variable which will tell us if the "RaiseEvent" was called or not.
var testRaiseEventWasCalled bool

func (streamProducerSender *MockEventStreamSender) RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error {
	testRaiseEventWasCalled = true

	return testRaiseEventData(eventType, payload)
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

	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageApplication, t
	}

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

	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageEndpoint, t
	}

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

	getStatusMessageAndTestUtility = func() (types.StatusMessage, *testing.T) {
		return statusMessageEndpoint, t
	}

	avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: endpointTestDataNotFound.MessageHeaders})

	if testRaiseEventWasCalled != endpointTestDataNotFound.RaiseEventCalled {
		t.Errorf(`Was RaiseEvent called? Want: "%t", got "%t"`, endpointTestDataNotFound.RaiseEventCalled, testRaiseEventWasCalled)
	}
}
