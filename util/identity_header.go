package util

import (
	"encoding/base64"
	"encoding/json"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// GeneratedXRhIdentity returns a base64 encoded header to use as x-rh-identity when one is not provided
func GeneratedXRhIdentity(account, orgId string) string {
	id := identity.XRHID{Identity: identity.Identity{
		AccountNumber: account,
		OrgID:         orgId},
	}
	bytes, err := json.Marshal(id)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(bytes)
}
