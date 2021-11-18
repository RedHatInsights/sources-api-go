package model

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type EventModelDao interface {
	BulkMessage(id *int64) (map[string]interface{}, error)
	FetchAndUpdateBy(id *int64, updateAttributes map[string]interface{}) error
	ToEventJSON(id *int64) ([]byte, error)
}

func UpdateMessage(eventObject *EventModelDao, resourceID int64, resourceType string, attributes []string) ([]byte, error) {
	updatedAttributes := map[string]interface{}{}
	updatedAttributes["updated"] = map[string]interface{}{resourceType: map[string]interface{}{strconv.Itoa(int(resourceID)): attributes}}

	bulkMessage, errorBulkMessage := (*eventObject).BulkMessage(&resourceID)
	if errorBulkMessage != nil {
		return nil, fmt.Errorf("error in BulkMessage: %s", errorBulkMessage.Error())
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
