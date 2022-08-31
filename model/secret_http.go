package model

type SecretResponse struct {
	ID string `json:"id"`

	Name     string                 `json:"name,omitempty"`
	AuthType string                 `json:"authtype"`
	Username string                 `json:"username"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}
