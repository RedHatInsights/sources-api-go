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

// IdentityFromKafkaHeaders returns an identity from the provided Kafka headers, if the array contains one of the
// "x-rh-sources-account-number" or "x-rh-identity" headers. It returns early on the first match, without any specific
// preference or order.
func IdentityFromKafkaHeaders(headers []kafka.Header) (*identity.Identity, error) {
	for _, header := range headers {
		if header.Key == xrhAccountNumberKey {
			return &identity.Identity{AccountNumber: string(header.Value)}, nil
		}

		if header.Key == xrhIdentityKey {
			xRhIdentity, err := ParseXRHIDHeader(string(header.Value))
			if err != nil {
				return nil, err
			}

			return &xRhIdentity.Identity, nil
		}
	}

	return nil, fmt.Errorf("unable to get identity number from headers, %s and %s are missing", xrhAccountNumberKey, xrhIdentityKey)
}
