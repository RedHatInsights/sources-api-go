package middleware

import (
	"encoding/json"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// middleware functions for each event type - adding this to the middleware
// stack will raise an event based on the `resource` field set in the context
// during the handler

var (
	RaiseSourceCreateEvent  = raiseEvent("Source.create")
	RaiseSourceUpdateEvent  = raiseEvent("Source.update")
	RaiseSourceDestroyEvent = raiseEvent("Source.destroy")

	RaiseApplicationCreateEvent  = raiseEvent("Application.create")
	RaiseApplicationUpdateEvent  = raiseEvent("Application.update")
	RaiseApplicationDestroyEvent = raiseEvent("Application.destroy")

	RaiseEndpointCreateEvent  = raiseEvent("Endpoint.create")
	RaiseEndpointUpdateEvent  = raiseEvent("Endpoint.update")
	RaiseEndpointDestroyEvent = raiseEvent("Endpoint.destroy")

	RaiseAuthenticationCreateEvent  = raiseEvent("Authentication.create")
	RaiseAuthenticationUpdateEvent  = raiseEvent("Authentication.update")
	RaiseAuthenticationDestroyEvent = raiseEvent("Authentication.destroy")

	RaiseApplicationAuthenticationCreateEvent  = raiseEvent("ApplicationAuthentication.create")
	RaiseApplicationAuthenticationUpdateEvent  = raiseEvent("ApplicationAuthentication.update")
	RaiseApplicationAuthenticationDestroyEvent = raiseEvent("ApplicationAuthentication.destroy")
)

// producer instance used to send messages - default just an empty instance of
// the struct.
var producer = events.EventStreamProducer{Sender: &events.EventStreamSender{}}

/*
	Function that takes an event-type as a string and then returns a middleware
	function that raises the specified event if the handler operation succeeds.

	It bails out if there isn't a `resource` field on the context which should
	be a model.ToEvent() call in the handler.
*/
func raiseEvent(eventType string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// first call the handler function (or the next middlware)
			err := next(c)
			if err != nil {
				return err
			}

			// pull the "event" resource from the context, which needs to be set
			// in the handler for this to work.
			resource := c.Get("resource")
			if resource == nil {
				l.Log.Infof("failed to pull event resource from context - skipping raise event")
				return nil
			}

			// specifically skip raising an event if this is set - usually when
			// a create action happened but we do not want to re-raise the
			// event.
			if c.Get("skip_raise") != nil {
				l.Log.Infof("skipping raise event per skip_raise set on context")
				return nil
			}

			l.Log.Infof("Raising Event %v", eventType)

			msg, err := json.Marshal(resource)
			if err != nil {
				return err
			}

			// TODO: make this async? Run this in a goroutine that way the
			// request isn't effectively blocked by kafka .
			headers := append(getRequestHeaders(c), kafka.Header{Key: "event_type", Value: []byte(eventType)})
			err = producer.RaiseEvent(eventType, msg, headers)
			if err != nil {
				l.Log.Warnf("failed to raise event to kafka: %v", err)
				return nil
			}

			return nil
		}
	}
}

/*
    Fetch the headers from the requeset that are needed to forward along

	1. x-rh-identity -- a generated one if it wasn't passed along (e.g. psk)

	2. x-rh-sources-psk -- always passed if present, and used for generation.
*/
func getRequestHeaders(c echo.Context) []kafka.Header {
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
