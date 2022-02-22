package mappers

import (
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/datatypes"
)

func MapRowToRhcConnection(row map[string]interface{}) (*model.RhcConnection, error) {
	var rhcConnection model.RhcConnection
	rhcConnection.AvailabilityStatus = model.AvailabilityStatus{}

	if value, ok := row["id"]; ok {
		if id, ok := value.(int64); ok {
			rhcConnection.ID = id
		}
	}

	if value, ok := row["rhc_id"]; ok {
		if rhcId, ok := value.(string); ok {
			rhcConnection.RhcId = rhcId
		}
	}

	if value, ok := row["extra"]; ok {
		if extra, ok := value.(string); ok {
			rhcConnection.Extra = datatypes.JSON(extra)
		}
	}

	if value, ok := row["availability_status"]; ok {
		if availabilityStatus, ok := value.(string); ok {
			rhcConnection.AvailabilityStatus.AvailabilityStatus = availabilityStatus
		}
	}

	if value, ok := row["availability_status_error"]; ok {
		if availabilityStatusError, ok := value.(string); ok {
			rhcConnection.AvailabilityStatusError = availabilityStatusError
		}
	}

	if value, ok := row["last_checked_at"]; ok {
		if lastCheckedAtStr, ok := value.(string); ok {
			lastCheckedAt, err := time.Parse(time.RFC3339, lastCheckedAtStr)
			if err != nil {
				return nil, err
			}

			rhcConnection.AvailabilityStatus.LastCheckedAt = lastCheckedAt
		}
	}

	if value, ok := row["last_available_at"]; ok {
		if lastAvailableAtStr, ok := value.(string); ok {
			lastAvailableAt, err := time.Parse(time.RFC3339, lastAvailableAtStr)
			if err != nil {
				return nil, err
			}

			rhcConnection.AvailabilityStatus.LastAvailableAt = lastAvailableAt
		}
	}

	if value, ok := row["created_at"]; ok {
		if createdAtStr, ok := value.(string); ok {
			createdAt, err := time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				return nil, err
			}

			rhcConnection.CreatedAt = createdAt
		}
	}

	if value, ok := row["updated_at"]; ok {
		if updatedAtStr, ok := value.(string); ok {
			updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				return nil, err
			}
			rhcConnection.UpdatedAt = updatedAt
		}
	}

	if value, ok := row["source_ids"]; ok {
		if idList, ok := value.(string); ok {
			err := MapIdListToRhcConnection(idList, &rhcConnection)
			if err != nil {
				return nil, err
			}
		}
	}

	return &rhcConnection, nil
}

// MapIdListToRhcConnection map a list of IDs given as a string separated by commas and creates a new source which then
// gets appended to the list of sources of the provided rhcConnection.
func MapIdListToRhcConnection(idListRaw string, connection *model.RhcConnection) error {
	if idListRaw == "" {
		return nil
	}

	ids := strings.Split(idListRaw, ",")

	// Loop through the ids and create sources accordingly to attach them to the connection.
	for _, strId := range ids {
		// Remove whitespaces in case the database returns them.
		strId = strings.TrimSpace(strId)

		id, err := strconv.ParseInt(strId, 10, 64)
		if err != nil {
			return err
		}

		connection.Sources = append(connection.Sources, model.Source{ID: id})
	}

	return nil
}
