package util

import (
	"encoding/base64"
	"encoding/json"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

// XRhIdentityWithAccountNumber returns a base64 encoded header to use as x-rh-identity when one is not provided
func XRhIdentityWithAccountNumber(account string) string {
	bytes, err := json.Marshal(identity.XRHID{Identity: identity.Identity{AccountNumber: account}})
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(bytes)
}
