package service

import (
	"encoding/json"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// Producer instance used to send messages - default just an empty instance of the struct.
var Producer = events.EventStreamProducer{Sender: &events.EventStreamSender{}}

// RaiseEvent raises an event with the provided resource.
func RaiseEvent(eventType string, resource model.Event, headers []kafka.Header) error {
	msg, err := json.Marshal(resource.ToEvent())
	if err != nil {
		return err
	}

	// TODO: make this async? Run this in a goroutine that way the
	// request isn't effectively blocked by kafka .
	headers = append(headers, kafka.Header{Key: "event_type", Value: []byte(eventType)})
	err = Producer.RaiseEvent(eventType, msg, headers)
	if err != nil {
		l.Log.Warnf("failed to raise event to kafka: %v", err)
		return nil
	}

	return nil
}

// ForwadableHeeaders fetches the required identity headers from the request that are needed to forward along:
// 	1. x-rh-identity -- a generated one if it wasn't passed along (e.g. psk)
//	2. x-rh-sources-psk -- always passed if present, and used for generation.
func ForwadableHeeaders(c echo.Context) []kafka.Header {
	headers := make([]kafka.Header, 0)

	if c.Get("psk-account") != nil {
		psk, ok := c.Get("psk-account").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-sources-account-number", Value: []byte(psk)})
		}
	}

	if c.Get("x-rh-identity") != nil {
		xrhid, ok := c.Get("x-rh-identity").(string)
		if ok {
			headers = append(headers, kafka.Header{Key: "x-rh-identity", Value: []byte(xrhid)})
		}
	} else {
		psk, ok := c.Get("psk-account").(string)
		if ok {
			// the only way this would be nil is if psk auth was used - so lets
			// generate a dummy header for services that still rely on it.
			headers = append(headers, kafka.Header{
				Key:   "x-rh-identity",
				Value: []byte(util.XRhIdentityWithAccountNumber(psk)),
			})
		}
	}

	return headers
}
