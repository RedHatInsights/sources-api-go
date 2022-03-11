package types

type StatusMessage struct {
	ResourceType  string      `json:"resource_type"`
	ResourceIDRaw interface{} `json:"resource_id"`
	ResourceID    string      `json:"-"`
	Status        string      `json:"status"`
	Error         string      `json:"error"`
}
