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
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

var testData []TestData

type MockFormatter struct {
	Hostname              string
	AppName               string
	InjectedToOtherLogger bool
}

func (m MockFormatter) Format(_ *logrus.Entry) ([]byte, error) {
	return []byte{}, nil
}

type MockEventStreamSender struct {
	events.EventStreamSender
	TestSuite *testing.T
	StatusMessage

	RaiseEventCalled bool
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
		src = source
	case "Application":
		application := &m.Application{ID: id}
		if forBulkMessage {
			res = dao.DB.Preload("Source").Preload("Source.Applications").Preload("Source.Endpoints")
		}

		res = res.Find(application)
		bulkMessage["applications"] = application.Source.Applications
		bulkMessage["endpoints"] = application.Source.Endpoints
		src = application
	case "Endpoint":
		endpoint := &m.Endpoint{ID: id}
		if forBulkMessage {
			res = dao.DB.Preload("Source").Preload("Source.Applications").Preload("Source.Endpoints")
		}
		res = res.Find(endpoint)
		bulkMessage["applications"] = endpoint.Source.Applications
		bulkMessage["endpoints"] = endpoint.Source.Endpoints
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

func (streamProducerSender *MockEventStreamSender) RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error {
	streamProducerSender.RaiseEventCalled = true
	var err error

	for _, data := range testData {
		if streamProducerSender.ResourceType == data.ResourceType && streamProducerSender.ResourceID == data.ResourceID {
			var isResult bool
			var expectedData []byte
			if eventType == "Records.update" {
				expectedData = BulkMessageFor(streamProducerSender.ResourceType, streamProducerSender.ResourceID)
			} else {
				expectedData = ResourceJSONFor(streamProducerSender.ResourceType, streamProducerSender.ResourceID)
			}

			isResult, err = JSONBytesEqual(payload, expectedData)
			if isResult != true {
				errMsg := "error in raising event of type %s with resource type %s.\n" +
					"JSON payloads are not same:\n\n" +
					"Expected: %s \n" +
					"Obtained: %s \n"

				streamProducerSender.TestSuite.Errorf(errMsg, eventType, streamProducerSender.ResourceType, expectedData, payload)
				return fmt.Errorf(errMsg, eventType, streamProducerSender.ResourceType, expectedData, payload)
			}
		}
	}

	if err != nil {
		streamProducerSender.TestSuite.Errorf("error with parsing JSON: %s", err.Error())
	}

	return err
}

type ExpectedData struct {
	BulkMessage  []byte
	ResourceJSON []byte
}

type TestData struct {
	StatusMessage
	m.AvailabilityStatus
	ExpectedData

	MessageHeaders []kafkaGo.Header

	RaiseEventCalled bool
}

func TestConsumeStatusMessage(t *testing.T) {
	if !runningIntegration {
		return
	}

	log := logrus.Logger{
		Out:          os.Stdout,
		Level:        logrus.DebugLevel,
		Formatter:    MockFormatter{},
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: false,
	}

	logging.Log = &log

	header := kafkaGo.Header{Key: "event_type", Value: []byte("availability_status")}
	header2 := kafkaGo.Header{Key: "x-rh-identity", Value: []byte("Test identity")}
	header3 := kafkaGo.Header{Key: "x-rh-sources-account-number", Value: []byte("12345")}
	headers := []kafkaGo.Header{header, header2, header3}
	statusMessage := StatusMessage{ResourceType: "Source", ResourceID: "1", Status: m.Available}
	sourceTestData := TestData{StatusMessage: statusMessage, MessageHeaders: headers, RaiseEventCalled: true}

	statusMessageApplication := StatusMessage{ResourceType: "Application", ResourceID: "1", Status: m.Available}
	applicationTestData := TestData{StatusMessage: statusMessageApplication, MessageHeaders: headers, RaiseEventCalled: true}

	statusMessageEndpoint := StatusMessage{ResourceType: "Endpoint", ResourceID: "1", Status: m.Available}
	endpointTestData := TestData{StatusMessage: statusMessageEndpoint, MessageHeaders: headers, RaiseEventCalled: true}

	statusMessageEndpoint = StatusMessage{ResourceType: "Endpoint", ResourceID: "99", Status: m.Available}
	endpointTestDataNotFound := TestData{StatusMessage: statusMessageEndpoint, MessageHeaders: headers, RaiseEventCalled: false}

	testData = make([]TestData, 4)
	testData[0] = sourceTestData
	testData[1] = applicationTestData
	testData[2] = endpointTestData
	testData[3] = endpointTestDataNotFound

	for _, testEntry := range testData {
		sender := MockEventStreamSender{TestSuite: t, StatusMessage: testEntry.StatusMessage}
		esp := &events.EventStreamProducer{Sender: &sender}
		avs := AvailabilityStatusListener{EventStreamProducer: esp}
		message, _ := json.Marshal(testEntry)
		avs.ConsumeStatusMessage(kafka.Message{Value: message, Headers: testEntry.MessageHeaders})

		raiseEventCalled := esp.Sender.(*MockEventStreamSender).RaiseEventCalled
		if raiseEventCalled != testEntry.RaiseEventCalled {
			wasOrWasNot := " "
			if raiseEventCalled == false {
				wasOrWasNot = " not "
			}

			wasOrWasNotExpected := " "
			if testEntry.RaiseEventCalled == false {
				wasOrWasNotExpected = " not "
			}

			sender.TestSuite.Errorf("RaiseEvent was%scalled while it was%sexpected", wasOrWasNot, wasOrWasNotExpected)
		}
	}
}
