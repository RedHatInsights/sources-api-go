package util

import (
	"encoding/base64"
	"encoding/json"
)

type minimalIdentityHeader struct {
	Identity id `json:"identity"`
}

type id struct {
	Account string `json:"account_number"`
}

func newMinimalIdentity(account string) *minimalIdentityHeader {
	return &minimalIdentityHeader{Identity: id{Account: account}}
}

// returns a base64 encoded header to use as x-rh-identity when one is not
// provided
func XRhIdentityWithAccountNumber(account string) string {
	bytes, err := json.Marshal(newMinimalIdentity(account))
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(bytes)
}
