package model

type ApplicationAuthenticationResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	ApplicationID     string `json:"application_id"`
	AuthenticationID  string `json:"authentication_id"`
	AuthenticationUID string `json:"authentication_uid"`
}
