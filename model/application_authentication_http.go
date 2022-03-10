package model

type ApplicationAuthenticationCreateRequest struct {
	ApplicationIDRaw    interface{} `json:"application_id"`
	AuthenticationIDRaw interface{} `json:"authentication_id"`
	ApplicationID       int64       `json:"-"`
	AuthenticationID    int64       `json:"-"`
}

type ApplicationAuthenticationResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	ApplicationID     string `json:"application_id"`
	AuthenticationID  string `json:"authentication_id"`
	AuthenticationUID string `json:"authentication_uid"`
}
