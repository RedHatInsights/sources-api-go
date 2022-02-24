package model

/*
	Bulk Create Request is a request creating 1..n resources in Sources API
	which does all of the linking between resources automatically.

	This outer request just contains slices of the possible resources that can
	be created.
*/
type BulkCreateRequest struct {
	Sources         []BulkCreateSource         `json:"sources"`
	Applications    []BulkCreateApplication    `json:"applications"`
	Endpoints       []BulkCreateEndpoint       `json:"endpoints"`
	Authentications []BulkCreateAuthentication `json:"authentications"`
}

type BulkCreateResponse struct {
	Sources         []SourceResponse         `json:"sources"`
	Applications    []ApplicationResponse    `json:"applications"`
	Endpoints       []EndpointResponse       `json:"endpoints"`
	Authentications []AuthenticationResponse `json:"authentications"`
}

////////////////////////////////////////////////////////////////////////////////
//
// For each of the resource types we're using struct embedding to re-use the
// CreateRequest models, with the extra bulk_create fields to look up foreign
// keys by name instead of needing primary keys for the *Type tables.
//
////////////////////////////////////////////////////////////////////////////////

type BulkCreateSource struct {
	SourceCreateRequest

	SourceTypeName string `json:"source_type_name"`
}

type BulkCreateApplication struct {
	ApplicationCreateRequest

	ApplicationTypeName string `json:"application_type_name"`
	SourceName          string `json:"source_name"`
}

type BulkCreateEndpoint struct {
	EndpointCreateRequest

	SourceName string `json:"source_name"`
}

type BulkCreateAuthentication struct {
	AuthenticationCreateRequest

	ResourceName string `json:"resource_name"`
}

/*
	Output from the BulkCreate operation - stores the parsed resources that have been validated.
*/
type BulkCreateOutput struct {
	Sources                    []Source
	Applications               []Application
	Endpoints                  []Endpoint
	Authentications            []Authentication
	ApplicationAuthentications []ApplicationAuthentication
}

func (b BulkCreateOutput) ToResponse() *BulkCreateResponse {
	resp := BulkCreateResponse{
		Sources:         make([]SourceResponse, len(b.Sources)),
		Applications:    make([]ApplicationResponse, len(b.Applications)),
		Endpoints:       make([]EndpointResponse, len(b.Endpoints)),
		Authentications: make([]AuthenticationResponse, len(b.Authentications)),
	}

	for i := 0; i < len(b.Sources); i++ {
		resp.Sources[i] = *b.Sources[i].ToResponse()
	}

	for i := 0; i < len(b.Applications); i++ {
		resp.Applications[i] = *b.Applications[i].ToResponse()
	}

	for i := 0; i < len(b.Sources[i].Endpoints); i++ {
		resp.Endpoints[i] = *b.Endpoints[i].ToResponse()
	}

	for i := 0; i < len(b.Authentications); i++ {
		resp.Authentications[i] = *b.Authentications[i].ToResponse()
	}

	return &resp
}
