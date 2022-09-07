package dao

// type aliases to make reading easier
type (
	sourceTypeSeedMap       map[string]sourceTypeSeed
	applicationTypeSeedMap  map[string]applicationTypeSeed
	superkeyMetadataSeedMap map[string]superKeySeed
	appMetadataSeedMap      map[string]appMetaDataSeed
)

type sourceTypeSeed struct {
	Category    string      `json:"category"`
	ProductName string      `json:"product_name"`
	Schema      interface{} `json:"schema"`
	Vendor      string      `json:"vendor"`
	IconURL     string      `json:"icon_url"`
}

type applicationTypeSeed struct {
	DisplayName                  string      `json:"display_name"`
	DependentApplications        interface{} `json:"dependent_applications"`
	SupportedSourceTypes         interface{} `json:"supported_source_types"`
	SupportedAuthenticationTypes interface{} `json:"supported_authentication_types"`
	ResourceOwnership            *string     `json:"resource_ownership"`
}

const AppMetaData = "AppMetaData"

type superKeySeed struct {
	Steps []superKeyStep `json:"steps"`
}

type superKeyStep struct {
	Step          int         `json:"step"`
	Name          string      `json:"name"`
	Payload       interface{} `json:"payload"`
	Substitutions interface{} `json:"substitutions"`
}

// it's just key values all the way down
type appMetaDataSeed map[string]map[string]string
