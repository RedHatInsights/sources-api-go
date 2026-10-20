package model

type EndpointResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	PausedAt  string `json:"paused_at,omitempty"`

	Role                    *string `json:"role,omitempty"`
	Port                    *int    `json:"port,omitempty"`
	Default                 *bool   `json:"default,omitempty"`
	Scheme                  *string `json:"scheme,omitempty"`
	Host                    *string `json:"host,omitempty"`
	Path                    *string `json:"path,omitempty"`
	VerifySsl               *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority    *string `json:"certificate_authority,omitempty"`
	ReceptorNode            *string `json:"receptor_node,omitempty"`
	AvailabilityStatus      *string `json:"availability_status,omitempty"`
	LastCheckedAt           string  `json:"last_checked_at,omitempty"`
	LastAvailableAt         string  `json:"last_available_at,omitempty"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`

	SourceID string `json:"source_id"`
}

type EndpointCreateRequest struct {
	Default              bool        `json:"default"`
	ReceptorNode         *string     `json:"receptor_node"`
	Role                 string      `json:"role"`
	Scheme               *string     `json:"scheme"`
	Host                 string      `json:"host"`
	Port                 *int        `json:"port"`
	Path                 string      `json:"path"`
	VerifySsl            *bool       `json:"verify_ssl"`
	CertificateAuthority *string     `json:"certificate_authority"`
	AvailabilityStatus   string      `json:"availability_status"`
	SourceID             int64       `json:"-"`
	SourceIDRaw          interface{} `json:"source_id"`
}

type EndpointEditRequest struct {
	Default                 *bool   `json:"default,omitempty"`
	ReceptorNode            *string `json:"receptor_node,omitempty"`
	Role                    *string `json:"role,omitempty"`
	Scheme                  *string `json:"scheme,omitempty"`
	Host                    *string `json:"host,omitempty"`
	Port                    *int    `json:"port,omitempty"`
	Path                    *string `json:"path,omitempty"`
	VerifySsl               *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority    *string `json:"certificate_authority,omitempty"`
	AvailabilityStatus      *string `json:"availability_status,omitempty"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`

	// TODO: remove these once satellite goes away.
	LastCheckedAt   *string `json:"last_checked_at,omitempty"`
	LastAvailableAt *string `json:"last_available_at,omitempty"`
}
