package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func ParseXRHIDHeader(inputIdentity string) (XRHIdentity *identity.XRHID, err error) {
	decodedIdentity, err := base64.StdEncoding.DecodeString(inputIdentity)
	if err != nil {
		return nil, fmt.Errorf("error decoding Identity: %v", err)
	}

	err = json.Unmarshal(decodedIdentity, &XRHIdentity)
	if err != nil {
		return nil, fmt.Errorf("x-rh-identity header does not contain valid JSON")
	}

	return XRHIdentity, nil
}

func AccountNumberFrom(headers []kafka.Header) (string, error) {
	XRHAccountNumberKey := "x-rh-sources-account-number"
	XRHIdentityKey := "x-rh-identity"

	for _, header := range headers {
		if header.Key == XRHAccountNumberKey {
			return string(header.Value), nil
		}

		if header.Key == XRHIdentityKey {
			XRHIdentity, err := ParseXRHIDHeader(string(header.Value))
			if err != nil {
				return "", err
			}

			return XRHIdentity.Identity.AccountNumber, nil
		}
	}

	return "", fmt.Errorf("unable to get account number from headers, %s and %s are missing", XRHAccountNumberKey, XRHIdentityKey)
}
