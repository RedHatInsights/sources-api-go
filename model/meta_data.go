package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type MetaData struct {
	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Step          int64          `json:"step"`
	Name          string         `json:"name"`
	Payload       datatypes.JSON `json:"payload"`
	Substitutions datatypes.JSON `json:"substitutions"`
	Type          string         `json:"type"`

	ApplicationTypeID int64 `json:"application_type_id"`
	ApplicationType   ApplicationType
}

func (app *MetaData) RelationInfo() map[string]RelationSetting {
	return map[string]RelationSetting{}
}

func (app *MetaData) ToResponse() *MetaDataResponse {
	id := strconv.FormatInt(app.ID, 10)
	appTypeId := strconv.FormatInt(app.ApplicationTypeID, 10)

	return &MetaDataResponse{
		ID:                id,
		CreatedAt:         util.DateTimeToRecordFormat(app.CreatedAt),
		UpdatedAt:         util.DateTimeToRecordFormat(app.UpdatedAt),
		Name:              app.Name,
		Payload:           app.Payload,
		ApplicationTypeId: appTypeId,
	}
}
