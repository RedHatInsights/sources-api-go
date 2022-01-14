package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// storing the satellite topic here since it doesn't change after initial
// startup.
var satelliteTopic = config.Get().KafkaTopic("platform.topological-inventory.operations-satellite")

// requests both types of availability checks for a source
func RequestAvailabilityCheck(source *m.Source) {
	l.Log.Infof("Requesting Availability Check for Source [%v]", source.ID)

	if len(source.Applications) != 0 {
		applicationAvailabilityCheck(source)
	}

	if len(source.Endpoints) != 0 {
		endpointAvailabilityCheck(source)
	}

	l.Log.Infof("Finished Publishing Availability Messages for Source %v", source.ID)
}

// sends off an availability check http request for each of the source's
// applications
func applicationAvailabilityCheck(source *m.Source) {
	for _, app := range source.Applications {
		l.Log.Infof("Requesting Availability Check for Application %v", app.ID)

		uri := app.ApplicationType.AvailabilityCheckURL()
		if uri == nil {
			l.Log.Warnf("Failed to fetch availability check url for [%v] - continuing", app.ApplicationType.Name)
			continue
		}

		requestAvailabilityCheck(source, &app, uri)
	}
}

func requestAvailabilityCheck(source *m.Source, app *m.Application, uri *url.URL) {
	httpClient := http.Client{Timeout: 10 * time.Second}

	body := map[string]string{"source_id": strconv.FormatInt(app.SourceID, 10)}
	raw, err := json.Marshal(body)
	if err != nil {
		l.Log.Warnf("Failed to marshal source body for [%v] - continuing", app.SourceID)
		return
	}

	req, err := http.NewRequest(http.MethodPost, uri.String(), bytes.NewBuffer(raw))
	if err != nil {
		l.Log.Warnf("Failed to make request for application [%v], uri [%v]", app.ID, uri.String())
		return
	}

	req.Header.Add("x-rh-sources-account-number", source.Tenant.ExternalTenant)
	req.Header.Add("x-rh-identity", util.XRhIdentityWithAccountNumber(source.Tenant.ExternalTenant))
	req.Header.Add("Content-Type", "application/json;charset=utf-8")

	resp, err := httpClient.Do(req)
	if err != nil {
		l.Log.Warnf("Error requesting availability status for application [%v], error: %v", app.ID, err)
		return
	}
	defer resp.Body.Close()

	// anything greater than 299 is bad, right??? right????
	if resp.StatusCode%100 > 2 {
		l.Log.Warnf("Bad response from client: %v", resp.StatusCode)
	}
}

// codified version of what we were sending over kafka. The satellite operations
// worker picks this message up and makes the proper requests to the
// platform-receptor-controller.
type satelliteAvailabilityMessage struct {
	SourceID       string  `json:"source_id"`
	SourceUID      *string `json:"source_uid"`
	SourceRef      *string `json:"source_ref"`
	ExternalTenant string  `json:"external_tenant"`
}

// sends off an availability check kafka message for each of the source's
// endpoints but only if the source is of type satellite - we do not support any
// other operations currently (legacy behavior)
func endpointAvailabilityCheck(source *m.Source) {
	if source.SourceType.Name != "satellite" {
		l.Log.Infof("Skipping Endpoint availability check for non-satellite source type")
		return
	}

	// instantiate a producer for this source
	mgr := &kafka.Manager{Config: kafka.Config{
		KafkaBrokers:   config.Get().KafkaBrokers,
		ProducerConfig: kafka.ProducerConfig{Topic: satelliteTopic},
	}}

	l.Log.Infof("Publishing message for Source [%v] topic [%v] ", source.ID, mgr.ProducerConfig.Topic)
	for _, endpoint := range source.Endpoints {
		publishSatelliteMessage(mgr, source, &endpoint)
	}
}

func publishSatelliteMessage(mgr *kafka.Manager, source *m.Source, endpoint *m.Endpoint) {
	l.Log.Infof("Requesting Availability Check for Endpoint %v", endpoint.ID)

	msg := &kafka.Message{}
	err := msg.AddValueAsJSON(&satelliteAvailabilityMessage{
		SourceID:       strconv.FormatInt(source.ID, 10),
		SourceUID:      source.Uid,
		SourceRef:      source.SourceRef,
		ExternalTenant: source.Tenant.ExternalTenant,
	})
	if err != nil {
		l.Log.Warnf("Failed to add struct value as json to kafka message")
		return
	}

	msg.AddHeaders([]kafka.Header{
		{Key: "x-rh-identity", Value: []byte(util.XRhIdentityWithAccountNumber(endpoint.Tenant.ExternalTenant))},
	})

	err = mgr.Produce(msg)
	if err != nil {
		l.Log.Warnf("Failed to produce kafka message for Source %v, error: %v", source.ID, err)
	}
}
