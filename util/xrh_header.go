package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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
	var outputIdentity identity.Identity

	for _, header := range headers {
		if header.Key == h.AccountNumberKey {
			outputIdentity.AccountNumber = string(header.Value)
		}

		if header.Key == h.OrgIdKey {
			outputIdentity.OrgID = string(header.Value)
		}

		if header.Key == h.IdentityKey {
			xRhIdentity, err := ParseXRHIDHeader(string(header.Value))
			if err != nil {
				return nil, err
			}

			outputIdentity = xRhIdentity.Identity
		}
	}

	if outputIdentity.AccountNumber == "" && outputIdentity.OrgID == "" {
		return nil, fmt.Errorf("unable to get identity number from headers, %s and %s are missing", h.AccountNumberKey, h.IdentityKey)
	}

	return &outputIdentity, nil
}
