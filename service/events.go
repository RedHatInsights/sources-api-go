package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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
//	2. x-rh-sources-psk -- always passed if present, and used for generation.
//	3. x-rh-sources-org-id -- always passed if present, and used for generation.
func ForwadableHeaders(c echo.Context) ([]kafka.Header, error) {
	headers := make([]kafka.Header, 0)

	if c.Get("x-rh-sources-psk") != nil {
		psk, ok := c.Get("x-rh-sources-psk").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-sources-account-number", Value: []byte(psk)})
		}
	}

	if c.Get("x-rh-sources-account-number") != nil {
		psk, ok := c.Get("x-rh-sources-account-number").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-sources-account-number", Value: []byte(psk)})
		}
	}

	if c.Get("x-rh-sources-org-id") != nil {
		orgId, ok := c.Get("x-rh-sources-org-id").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-sources-org-id", Value: []byte(orgId)})
		}
	}

	if c.Get("x-rh-identity") != nil {
		xrhid, ok := c.Get("x-rh-identity").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-identity", Value: []byte(xrhid)})
		}
	} else {
		var xRhId identity.XRHID

		orgId, orgIdOk := c.Get("x-rh-sources-org-id").(string)
		if orgIdOk {
			xRhId.Identity.OrgID = orgId
		}

		psk, pskOk := c.Get("x-rh-sources-psk").(string)
		if pskOk {
			xRhId.Identity.AccountNumber = psk
		}

		if orgIdOk || pskOk {
			contents, err := json.Marshal(xRhId)
			if err != nil {
				logging.Log.Errorf(`[account_number: %s][org_id: %s] Could not marshal xRhId object: %s`, xRhId.Identity.AccountNumber, xRhId.Identity.OrgID, err)

				return nil, errors.New("error generating identity header")
			}

			encodedXrhId := base64.StdEncoding.EncodeToString(contents)
			headers = append(headers, kafka.Header{Key: "x-rh-identity", Value: []byte(encodedXrhId)})
		}

	}

	return headers, nil
}
