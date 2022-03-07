package statuslistener

import (
	"os"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/types"
	m "github.com/RedHatInsights/sources-api-go/model"
)

// runningIntegration is used to skip integration tests if we're just running unit tests.
var runningIntegration = false

func TestMain(t *testing.M) {
	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		runningIntegration = true
		database.ConnectAndMigrateDB("status_listener")
		database.CreateFixtures()
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("status_listener")
	}

	os.Exit(code)
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
