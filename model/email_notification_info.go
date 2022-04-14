package model

type EmailNotification interface {
	ToEmail(previousStatus string) *EmailNotificationInfo
}

type EmailNotificationInfo struct {
	ResourceDisplayName        string `json:"resource_display_name"`
	CurrentAvailabilityStatus  string `json:"current_availability_status"`
	PreviousAvailabilityStatus string `json:"previous_availability_status"`
	SourceID                   string `json:"source_id"`
	SourceName                 string `json:"source_name"`
}
