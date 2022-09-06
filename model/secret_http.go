package model

type SecretResponse struct {
	ID string `json:"id"`

	Name     string                 `json:"name,omitempty"`
	AuthType string                 `json:"authtype"`
	Username string                 `json:"username"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

type SecretInternalResponse struct {
	ID string `json:"id"`

	Name     string                 `json:"name,omitempty"`
	AuthType string                 `json:"authtype"`
	Username string                 `json:"username"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
	Password string                 `json:"password,omitempty"`
}

type SecretCreateRequest struct {
	Name       *string                `json:"name,omitempty"`
	AuthType   string                 `json:"authtype"`
	Username   *string                `json:"username"`
	Password   *string                `json:"password,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
	UserScoped bool                   `json:"user_scoped"`
}
