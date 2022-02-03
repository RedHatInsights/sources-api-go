package model

type EndpointResponse struct {
	AvailabilityStatusResponse
	PauseResponse

	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	Role                    *string `json:"role,omitempty"`
	Port                    *int    `json:"port,omitempty"`
	Default                 *bool   `json:"default,omitempty"`
	Scheme                  *string `json:"scheme,omitempty"`
	Host                    *string `json:"host,omitempty"`
	Path                    *string `json:"path,omitempty"`
	VerifySsl               *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority    *string `json:"certificate_authority,omitempty"`
	ReceptorNode            *string `json:"receptor_node,omitempty"`
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
