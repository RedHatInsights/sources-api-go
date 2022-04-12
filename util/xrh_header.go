package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const (
	xrhAccountNumberKey string = "x-rh-sources-account-number"
	xrhIdentityKey      string = "x-rh-identity"
)

func ParseXRHIDHeader(inputIdentity string) (*identity.XRHID, error) {
	var XRHIdentity *identity.XRHID
	decodedIdentity, err := base64.StdEncoding.DecodeString(inputIdentity)
	if err != nil {
		return nil, fmt.Errorf("error decoding Identity: %v", err)
	}

	err = json.Unmarshal(decodedIdentity, &XRHIdentity)
	if err != nil {
		logging.Log.Debugf("x-rh-identity header does not valid JSON: %s. Payload: %s", err, inputIdentity)
		return nil, fmt.Errorf("x-rh-identity header does not contain valid JSON: %s", err)
	}

	return XRHIdentity, nil
}

func AccountNumberFromHeaders(headers []kafka.Header) (string, error) {
	for _, header := range headers {
		if header.Key == xrhAccountNumberKey {
			return string(header.Value), nil
		}

		if header.Key == xrhIdentityKey {
			XRHIdentity, err := ParseXRHIDHeader(string(header.Value))
			if err != nil {
				return "", err
			}

			return XRHIdentity.Identity.AccountNumber, nil
		}
	}

	return "", fmt.Errorf("unable to get account number from headers, %s and %s are missing", xrhAccountNumberKey, xrhIdentityKey)
}
