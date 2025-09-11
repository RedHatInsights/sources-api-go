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
	// JWT-related: Authorization is actual HTTP header, others are internal context keys
	Authorization = "Authorization" // HTTP header: "Authorization: Bearer <token>"
	JWTToken      = "jwt-token"     // Context key: extracted JWT token string
	JWTIssuer     = "jwt-issuer"    // Context key: validated JWT issuer
	JWTSubject    = "jwt-subject"   // Context key: validated JWT subject
)
