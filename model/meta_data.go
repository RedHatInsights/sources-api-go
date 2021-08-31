package model

import (
	"strconv"
	"time"

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
	var settings = make(map[string]RelationSetting)

	settings["application_type"] = RelationSetting{RelationType: "HasMany"}

	return settings
}

func (app *MetaData) ToResponse() *MetaDataResponse {
	id := strconv.FormatInt(app.ID, 10)

	return &MetaDataResponse{
		ID:            id,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
		Step:          app.Step,
		Name:          app.Name,
		Payload:       app.Payload,
		Substitutions: app.Substitutions,
		Type:          app.Type,
	}
}
