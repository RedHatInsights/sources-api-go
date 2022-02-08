package middleware

import (
	"encoding/json"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
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
func RaiseEvent(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// first call the handler function (or the next middlware)
		err := next(c)
		if err != nil {
			return err
		}

		// specifically skip raising an event if this is set - usually when
		// a create action happened but we do not want to re-raise the
		// event.
		if c.Get("skip_raise") != nil {
			l.Log.Infof("skipping raise event per skip_raise set on context")
			return nil
		}

		// pull the "event" resource from the context, which needs to be set
		// in the handler for this to work.
		resource := c.Get("resource")
		if resource == nil {
			l.Log.Infof("failed to pull event resource from context - skipping raise event")
			return nil
		}

		eventType, ok := c.Get("event_type").(string)
		if !ok {
			l.Log.Warnf("Failed to cast event_type to string - exiting")
			return nil
		}

		if c.Get("event_override") != nil {
			event, ok := c.Get("event_override").(string)
			if !ok {
				l.Log.Warnf("Failed to cast event_override from request - ditching post to kafka")
				return nil
			}

			l.Log.Infof("Using overridden event_type %v instead of %v", c.Get("event_override"), eventType)
			eventType = event
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
