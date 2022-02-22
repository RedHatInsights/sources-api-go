package mappers

import (
	"bytes"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/model"
	"gorm.io/datatypes"
)

// validNumberOfSourceIds stores the total amount of source ids that the list has.
const validNumberOfSourceIds = 5

const validId = int64(1)
const validRhcId = "rhcIdUuid"
const validExtra = `{"hello": "world"}`
const validAvailabilityStatus = "available"
const validAvailabilityStatusError = ""
const validLastCheckedAt = "2000-01-01T00:00:00Z"
const validLastAvailableAt = "2001-01-01T00:00:00Z"
const validCreatedAt = "1998-01-01T00:00:00Z"
const validUpdatedAt = "1999-01-01T00:00:00Z"
const validSourceIdList = "1, 2, 3, 4, 5"

// setUpValidDatabaseRow sets up a valid database row the way the functions expect it.
func setUpValidDatabaseRow() map[string]interface{} {
	return map[string]interface{}{
		"id":                        validId,
		"rhc_id":                    validRhcId,
		"extra":                     validExtra,
		"availability_status":       validAvailabilityStatus,
		"availability_status_error": validAvailabilityStatusError,
		"last_checked_at":           validLastCheckedAt,
		"last_available_at":         validLastAvailableAt,
		"created_at":                validCreatedAt,
		"updated_at":                validUpdatedAt,
		"source_ids":                validSourceIdList,
	}
}

// TestMapRowToRhcConnection tests that a row is properly mapped to an rhcConnection.
func TestMapRowToRhcConnection(t *testing.T) {
	databaseRow := setUpValidDatabaseRow()

	result, err := MapRowToRhcConnection(databaseRow)
	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	{
		want := validId
		got := result.ID
		if want != got {
			t.Errorf(`Unexpected different ids found. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := validRhcId
		got := result.RhcId
		if want != got {
			t.Errorf(`Unexpected different rhcIds found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := datatypes.JSON(validExtra)
		got := result.Extra
		if !bytes.Equal(want, got) {
			t.Errorf(`Unexpected different extra fields found. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := validAvailabilityStatus
		got := result.AvailabilityStatus.AvailabilityStatus
		if want != got {
			t.Errorf(`Unexpected different availability statuses found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := validAvailabilityStatusError
		got := result.AvailabilityStatusError
		if want != got {
			t.Errorf(`Unexpected different availability status error found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want, err := time.Parse(time.RFC3339, validLastCheckedAt)
		if err != nil {
			t.Errorf("error parsing time: %s", err)
		}

		got := result.LastCheckedAt
		if want != got {
			t.Errorf(`Unexpected different last cheked at times found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want, err := time.Parse(time.RFC3339, validLastAvailableAt)
		if err != nil {
			t.Errorf("error parsing time: %s", err)
		}

		got := result.LastAvailableAt
		if want != got {
			t.Errorf(`Unexpected different last available times found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want, err := time.Parse(time.RFC3339, validCreatedAt)
		if err != nil {
			t.Errorf("error parsing time: %s", err)
		}

		got := result.CreatedAt
		if want != got {
			t.Errorf(`Unexpected different create times found. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want, err := time.Parse(time.RFC3339, validUpdatedAt)
		if err != nil {
			t.Errorf("error parsing time: %s", err)
		}

		got := result.UpdatedAt
		if want != got {
			t.Errorf(`Unexpected different update times found. Want "%s", got "%s"`, want, got)
		}
	}
}

// TestIdListToRhcConnection tests if the list of ids is correctly mapped to a slice of sources.
func TestIdListToRhcConnection(t *testing.T) {
	var rhcConnection model.RhcConnection

	err := MapIdListToRhcConnection(validSourceIdList, &rhcConnection)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}

	{
		want := validNumberOfSourceIds
		got := len(rhcConnection.Sources)
		if want != got {
			t.Errorf(`want "%d" soruces, got "%d"`, want, got)
		}
	}

	{
		for i := 0; i < validNumberOfSourceIds; i++ {
			want := int64(i + 1)
			got := rhcConnection.Sources[i].ID

			if want != got {
				t.Errorf(`source IDs don't match. Want "%d", got "%d"'`, want, got)
			}
		}
	}
}

// TestIdListEmpty tests that the mapping function doesn't break if the database returns an empty string.
func TestIdListEmpty(t *testing.T) {
	var rhcConnection model.RhcConnection

	err := MapIdListToRhcConnection("", &rhcConnection)

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}
}
