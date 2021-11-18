package statuslistener

type StatusMessage struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Status       string `json:"status"`
	Error        string `json:"error"`
}
