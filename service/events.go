package service

import (
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// Producer instance used to send messages - default just an empty instance of the struct.
var Producer = func() events.Sender { return events.EventStreamProducer{Sender: &events.EventStreamSender{}} }

// RaiseEvent raises an event with the provided resource.
func RaiseEvent(eventType string, resource model.Event, headers []kafka.Header) error {
	msg, err := json.Marshal(resource.ToEvent())
	if err != nil {
		return fmt.Errorf("failed to marshal %+v as event: %v", resource, err)
	}

	headers = append(headers, kafka.Header{Key: "event_type", Value: []byte(eventType)})
	err = Producer().RaiseEvent(eventType, msg, headers)
	if err != nil {
		return fmt.Errorf("failed to raise event to kafka: %v", err)
	}

	return nil
}

// ForwadableHeaders fetches the required identity headers from the request that are needed to forward along:
// 	1. x-rh-identity -- a generated one if it wasn't passed along (e.g. psk)
//	2. x-rh-sources-account-number -- always passed if present, and used for generation.
//	3. x-rh-sources-org-id -- always passed if present, and used for generation.
func ForwadableHeaders(c echo.Context) ([]kafka.Header, error) {
	headers := make([]kafka.Header, 0)
	var account, orgId, xrhid string
	var ok bool

	// pulling the specified account if it exists
	if c.Get(h.ACCOUNT_NUMBER) != nil {
		account, ok = c.Get(h.ACCOUNT_NUMBER).(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast psk-account to string")
		}
	}

	// pulling the specified orgId if it exists
	if c.Get(h.OrgID) != nil {
		orgId, ok = c.Get(h.OrgID).(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast psk-account to string")
		}
	}

	// pull the xrhid OR generate one using the information from the PSK information.
	if c.Get(h.XRHID) != nil {
		rhid, ok := c.Get(h.XRHID).(string)
		if ok {
			// set the xrhid to be passed on
			xrhid = rhid

			// parse the encoded identity in case the psk-fields weren't set.
			id, err := util.ParseXRHIDHeader(rhid)
			if err != nil {
				return nil, err
			}

			// account and orgId will be "" if they weren't present, so lets set them just in case.
			if account == "" {
				account = id.Identity.AccountNumber
			}
			if orgId == "" {
				orgId = id.Identity.OrgID
			}
		}
	} else {
		xrhid = util.GeneratedXRhIdentity(account, orgId)
	}

	// need to check org_id + account since one or the other might not be there.
	if account != "" {
		headers = append(headers, kafka.Header{Key: h.ACCOUNT_NUMBER, Value: []byte(account)})
	}
	if orgId != "" {
		headers = append(headers, kafka.Header{Key: h.OrgID, Value: []byte(orgId)})
	}

	headers = append(headers, kafka.Header{Key: h.XRHID, Value: []byte(xrhid)})

	return headers, nil
}
