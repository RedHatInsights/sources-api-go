package model

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/util"
)

type Event interface {
	ToEvent() interface{}
}

type EventModelDao interface {
	BulkMessage(resource util.Resource) (map[string]interface{}, error)
	FetchAndUpdateBy(resource util.Resource, updateAttributes map[string]interface{}) (interface{}, error)
	ToEventJSON(resource util.Resource) ([]byte, error)
}

func UpdateMessage(eventObject EventModelDao, resource util.Resource, attributes []string) ([]byte, error) {
	updatedAttributes := map[string]interface{}{}

	resourceID := ""
	if resource.ResourceUID != "" && resource.ResourceID == 0 {
		resourceID = resource.ResourceUID
	} else {
		resourceID = strconv.Itoa(int(resource.ResourceID))
	}

	updatedAttributes["updated"] = map[string]interface{}{resource.ResourceType: map[string]interface{}{resourceID: attributes}}

	bulkMessage, err := eventObject.BulkMessage(resource)
	if err != nil {
		return nil, fmt.Errorf("error in BulkMessage: %v", err.Error())
	}
	for k, m := range bulkMessage {
		updatedAttributes[k] = m
	}

	data, err := json.Marshal(updatedAttributes)
	if err != nil {
		return nil, fmt.Errorf("error in parsing update attributes: %s", err.Error())
	}

	return data, nil
}
