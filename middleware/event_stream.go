package middleware

import (
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

// RaiseEvent calls the "RaiseEvent" function once the previous handler has succeeded. It grabs the resource and the
// event type from the context.
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
			c.Logger().Infof("skipping raise event per skip_raise set on context")
			return nil
		}

		// pull the "event" resource from the context, which needs to be set
		// in the handler for this to work.
		resource, ok := c.Get("resource").(model.Event)
		if !ok {
			c.Logger().Infof("failed to pull event resource from context - skipping raise event")
			return nil
		}

		eventType, ok := c.Get("event_type").(string)
		if !ok {
			c.Logger().Warnf("Failed to cast event_type to string - exiting")
			return nil
		}

		if c.Get("event_override") != nil {
			event, ok := c.Get("event_override").(string)
			if !ok {
				c.Logger().Warnf("Failed to cast event_override from request - ditching post to kafka")
				return nil
			}

			c.Logger().Infof("Using overridden event_type %v instead of %v", c.Get("event_override"), eventType)
			eventType = event
		}

		c.Logger().Infof("Raising Event %v", eventType)

		headers, err := service.ForwadableHeaders(c)
		if err != nil {
			return err
		}

		// async!
		go func() {
			err := service.RaiseEvent(eventType, resource, headers)
			if err != nil {
				c.Logger().Warnf("Error raising event %v: %v", eventType, err)
			}
		}()

		return nil
	}
}
