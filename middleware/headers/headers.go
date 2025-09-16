package headers

const (
	PSK               = "x-rh-sources-psk"
	AccountNumber     = "x-rh-sources-account-number"
	OrgID             = "x-rh-sources-org-id"
	SkipEmptySources  = "x-rh-sources-skip-empty-sources"
	PSKUserID         = "x-rh-sources-user-id"
	XRHID             = "x-rh-identity"
	InsightsRequestID = "x-rh-insights-request-id"
	EdgeRequestID     = "x-rh-edge-request-id"
	ParsedIdentity    = "identity"
	TenantID          = "tenantID"
	UserID            = "userID"
	Authorization     = "Authorization" // HTTP header: "Authorization: Bearer <token>"
	JWTIssuer         = "jwt-issuer"    // Context key: verified JWT issuer
	JWTSubject        = "jwt-subject"   // Context key: verified JWT subject
)
