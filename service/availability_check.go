package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const (
	disconnectedRhc = "cloud-connector returned 'disconnected'"
	unavailbleRhc   = "cloud-connector returned a non-ok exit code for this connection"
)

type availabilityCheckRequester struct{}

type availabilityChecker interface {
	ApplicationAvailabilityCheck(source *m.Source)
	EndpointAvailabilityCheck(source *m.Source)
	RhcConnectionAvailabilityCheck(source *m.Source, headers []kafka.Header)
}

var (
	// storing the satellite topic here since it doesn't change after initial
	// startup.
	satelliteTopic = config.Get().KafkaTopic("platform.topological-inventory.operations-satellite")
	// default availability checker instance
	ac availabilityChecker = &availabilityCheckRequester{}
	// cloud connector related fields
	cloudConnectorUrl      = os.Getenv("CLOUD_CONNECTOR_AVAILABILITY_CHECK_URL")
	cloudConnectorPsk      = os.Getenv("CLOUD_CONNECTOR_PSK")
	cloudConnectorClientId = os.Getenv("CLOUD_CONNECTOR_CLIENT_ID")
)

// requests both types of availability checks for a source
func RequestAvailabilityCheck(source *m.Source, headers []kafka.Header) {
	l.Log.Infof("Requesting Availability Check for Source [%v]", source.ID)

	if len(source.Applications) != 0 {
		ac.ApplicationAvailabilityCheck(source)
	}

	if len(source.Endpoints) != 0 {
		ac.EndpointAvailabilityCheck(source)
	}

	if len(source.SourceRhcConnections) != 0 {
		ac.RhcConnectionAvailabilityCheck(source, headers)
	}

	l.Log.Infof("Finished Publishing Availability Messages for Source %v", source.ID)
}

// sends off an availability check http request for each of the source's
// applications
func (acr availabilityCheckRequester) ApplicationAvailabilityCheck(source *m.Source) {
	for _, app := range source.Applications {
		l.Log.Infof("Requesting Availability Check for Application %v", app.ID)

		uri := app.ApplicationType.AvailabilityCheckURL()
		if uri == nil {
			l.Log.Warnf("Failed to fetch availability check url for [%v] - continuing", app.ApplicationType.Name)
			continue
		}

		httpAvailabilityRequest(source, &app, uri)
	}
}

func httpAvailabilityRequest(source *m.Source, app *m.Application, uri *url.URL) {
	body := map[string]string{"source_id": strconv.FormatInt(app.SourceID, 10)}
	raw, err := json.Marshal(body)
	if err != nil {
		l.Log.Warnf("Failed to marshal source body for [%v] - continuing", app.SourceID)
		return
	}

	// spin up a 10 second context to limit the time spent waiting on a response
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewBuffer(raw))
	if err != nil {
		l.Log.Warnf("Failed to make request for application [%v], uri [%v]", app.ID, uri.String())
		return
	}

	req.Header.Add("x-rh-sources-org-id", source.Tenant.OrgID)
	req.Header.Add("x-rh-sources-account-number", source.Tenant.ExternalTenant)
	req.Header.Add("x-rh-identity", util.GeneratedXRhIdentity(source.Tenant.ExternalTenant, source.Tenant.OrgID))
	req.Header.Add("Content-Type", "application/json;charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.Log.Warnf("Error requesting availability status for application [%v], error: %v", app.ID, err)
		return
	}
	defer resp.Body.Close()

	// anything greater than 299 is bad, right??? right????
	if resp.StatusCode/100 > 2 {
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
func (acr availabilityCheckRequester) EndpointAvailabilityCheck(source *m.Source) {
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
	err := msg.AddValueAsJSON(map[string]interface{}{
		"params": satelliteAvailabilityMessage{
			SourceID:       strconv.FormatInt(source.ID, 10),
			SourceUID:      source.Uid,
			SourceRef:      source.SourceRef,
			ExternalTenant: source.Tenant.ExternalTenant,
		}})
	if err != nil {
		l.Log.Warnf("Failed to add struct value as json to kafka message")
		return
	}

	msg.AddHeaders([]kafka.Header{
		{Key: "event_type", Value: []byte("Source.availability_check")},
		{Key: "encoding", Value: []byte("json")},
		{Key: "x-rh-identity", Value: []byte(util.GeneratedXRhIdentity(source.Tenant.ExternalTenant, source.Tenant.OrgID))},
		{Key: "x-rh-sources-account-number", Value: []byte(endpoint.Tenant.ExternalTenant)},
	})

	err = mgr.Produce(msg)
	if err != nil {
		l.Log.Warnf("Failed to produce kafka message for Source %v, error: %v", source.ID, err)
	}

	err = mgr.Producer().Close()
	if err != nil {
		l.Log.Warnf("Failed to close kafka producer: %v", err)
	}
}

type rhcConnectionStatusResponse struct {
	Status string `json:"status"`
}

// hit the RHC connector running in-cluster in order to check and see if the
// status for each RHC id is connected or disconnected
func (acr availabilityCheckRequester) RhcConnectionAvailabilityCheck(source *m.Source, headers []kafka.Header) {
	for i := range source.SourceRhcConnections {
		go pingRHC(source, &source.SourceRhcConnections[i].RhcConnection, headers)
	}
}

func pingRHC(source *m.Source, rhcConnection *m.RhcConnection, headers []kafka.Header) {
	if cloudConnectorUrl == "" {
		l.Log.Warnf("CLOUD_CONNECTOR_AVAILABILITY_CHECK_URL not set - skipping check for RHC Connection Availability Status [%v]", rhcConnection.RhcId)
		return
	}

	l.Log.Infof("Requesting Availability Check for RHC %v", rhcConnection.ID)

	// per: https://github.com/RedHatInsights/cloud-connector/blob/master/internal/controller/api/api.spec.json
	body, err := json.Marshal(map[string]interface{}{
		"account": source.Tenant.ExternalTenant,
		"node_id": rhcConnection.RhcId,
	})
	if err != nil {
		l.Log.Warnf("Failed to marshal request body: %v", err)
		return
	}

	// timeout after 10s
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cloudConnectorUrl, bytes.NewBuffer(body))
	if err != nil {
		l.Log.Warnf("Failed to create request for RHC Connection for ID %v, e: %v", source.ID, err)
		return
	}
	req.Header.Set("x-rh-cloud-connector-org-id", source.Tenant.OrgID)
	req.Header.Set("x-rh-cloud-connector-account", source.Tenant.ExternalTenant)
	req.Header.Set("x-rh-cloud-connector-client-id", cloudConnectorClientId)
	req.Header.Set("x-rh-cloud-connector-psk", cloudConnectorPsk)

	// Log the request before sending it.
	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status request: %#v`, source.ID, rhcConnection.ID, rhcConnection.RhcId, req)
	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status request's body: %v`, source.ID, rhcConnection.ID, rhcConnection.RhcId, string(body))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.Log.Warnf("Failed to request connection_status for RHC ID [%v]: %v", rhcConnection.RhcId, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		l.Log.Warnf("Invalid return code received for RHC ID [%v]: %v", rhcConnection.RhcId, resp.StatusCode)
		b, _ := io.ReadAll(resp.Body)
		l.Log.Warnf("Body Returned from RHC ID [%v]: %s", rhcConnection.ID, b)

		// updating status to unavailable
		updateRhcStatus(source, "unavailable", unavailbleRhc, rhcConnection, headers)
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Log.Warnf("failed to read body from request: %v", err)
		return
	}

	// Log everything from the response
	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status received response: %#v`, source.ID, rhcConnection.ID, rhcConnection.RhcId, resp)
	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status response status code: %d`, source.ID, rhcConnection.ID, rhcConnection.RhcId, resp.StatusCode)
	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status response body: %s`, source.ID, rhcConnection.ID, rhcConnection.RhcId, b)

	var status rhcConnectionStatusResponse
	err = json.Unmarshal(b, &status)
	if err != nil {
		l.Log.Warnf("failed to unmarshal response: %v", err)
		return
	}

	var sanitizedStatus string
	var errstr string
	switch status.Status {
	case "connected":
		sanitizedStatus = "available"
	case "disconnected":
		sanitizedStatus = "unavailable"
		errstr = disconnectedRhc
	default:
		l.Log.Warnf("Invalid status returned from RHC: %v", status.Status)
		return
	}

	// only go through and update if there was a change.
	if rhcConnection.AvailabilityStatus != sanitizedStatus {
		updateRhcStatus(source, sanitizedStatus, errstr, rhcConnection, headers)
	}
}

func updateRhcStatus(source *m.Source, status string, errstr string, rhcConnection *m.RhcConnection, headers []kafka.Header) {
	now := time.Now()

	source.AvailabilityStatus = status
	source.LastCheckedAt = &now
	rhcConnection.AvailabilityStatus = status
	rhcConnection.LastCheckedAt = &now

	if status == m.Available {
		source.LastAvailableAt = &now
		rhcConnection.LastAvailableAt = &now
	} else {
		rhcConnection.AvailabilityStatusError = errstr
	}

	err := dao.GetSourceDao(&source.TenantID).Update(source)
	if err != nil {
		l.Log.Warnf("failed to update source availability status: %v", err)
		return
	}

	err = dao.GetRhcConnectionDao(&source.TenantID).Update(rhcConnection)
	if err != nil {
		l.Log.Warnf("failed to update RHC Connection availability status: %v", err)
		return
	}

	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d] RHC Connection's status updated to "%s"`, source.ID, rhcConnection.ID, status)

	err = RaiseEvent("RhcConnection.update", rhcConnection, headers)
	if err != nil {
		l.Log.Warnf("error raising RhcConnection.update event: %v", err)
	}
}
