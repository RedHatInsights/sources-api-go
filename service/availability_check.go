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

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/metrics"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
)

const (
	disconnectedRhc = "cloud-connector returned 'disconnected'"
	unavailableRhc  = "cloud-connector returned a non-ok exit code for this connection"
)

type availabilityCheckRequester struct {
	// storing the echo context so we can pull the logger
	c echo.Context
	// metricsService is used to count the successful and failed availability check requests sent to downstream
	// services.
	metricsService metrics.MetricsService
}

type availabilityChecker interface {
	// public methods
	ApplicationAvailabilityCheck(source *m.Source, skipEmptySources bool)
	RhcConnectionAvailabilityCheck(source *m.Source, headers []kafka.Header)

	// private methods
	httpAvailabilityRequest(source *m.Source, app *m.Application, uri *url.URL)
	pingRHC(source *m.Source, rhcConnection *m.RhcConnection, headers []kafka.Header)
	updateRhcStatus(source *m.Source, status string, errstr string, rhcConnection *m.RhcConnection, headers []kafka.Header)

	// le logger
	Logger() echo.Logger
}

var (
	// cloud connector related fields
	cloudConnectorUrl        = os.Getenv("CLOUD_CONNECTOR_AVAILABILITY_CHECK_URL")
	cloudConnectorStatusPath = os.Getenv("CLOUD_CONNECTOR_STATUS_PATH")
	cloudConnectorPsk        = os.Getenv("CLOUD_CONNECTOR_PSK")
	cloudConnectorClientId   = os.Getenv("CLOUD_CONNECTOR_CLIENT_ID")
)

// requests both types of availability checks for a source
func RequestAvailabilityCheck(metricsService metrics.MetricsService, c echo.Context, source *m.Source, headers []kafka.Header, skipEmptySources bool) {
	var ac availabilityChecker = &availabilityCheckRequester{metricsService: metricsService, c: c}
	ac.Logger().Infof("[source_id: %d] Requesting availability check for source", source.ID)

	if len(source.Applications) != 0 {
		ac.ApplicationAvailabilityCheck(source, skipEmptySources)
	}

	// we only want to send endpoint requests if we _do not_ have any endpoints
	// associated with this source. This way the satellite worker has no chance
	// of overwriting the status set by the RHC check
	if len(source.SourceRhcConnections) != 0 {
		ac.RhcConnectionAvailabilityCheck(source, headers)
	}

	ac.Logger().Infof("Finished Publishing Availability Messages for Source %v", source.ID)
}

// sends off an availability check http request for each of the source's
// applications
func (acr availabilityCheckRequester) ApplicationAvailabilityCheck(source *m.Source, skipEmptySources bool) {
	for _, app := range source.Applications {
		// We do not want to send availability check requests for Cost Management applications, as detailed in
		// https://issues.redhat.com/browse/RHCLOUD-38735.
		if skipEmptySources && isCostManagementApplication(app) {
			continue
		}

		uri := app.ApplicationType.AvailabilityCheckURL()
		if uri == nil {
			acr.Logger().Errorf("[source_id: %d][application_id: %d][application_type: %s] Failed to fetch availability check url - continuing", source.ID, app.ID, app.ApplicationType.Name)
			acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

			continue
		}

		acr.Logger().Infof("[source_id :%d][application_id: %d][uri: %s] Requesting availability check for application", source.ID, app.ID, uri)
		acr.httpAvailabilityRequest(source, &app, uri)
	}
}

func (acr availabilityCheckRequester) httpAvailabilityRequest(source *m.Source, app *m.Application, uri *url.URL) {
	body := map[string]string{"source_id": strconv.FormatInt(app.SourceID, 10)}

	raw, err := json.Marshal(body)
	if err != nil {
		acr.Logger().Errorf("[source_id: %d] Failed to marshal source body: %s", app.SourceID, err)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

		return
	}

	// spin up a 10 second context to limit the time spent waiting on a response
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewBuffer(raw))
	if err != nil {
		acr.Logger().Errorf("[source_id: %d][application_id: %d][uri: %s] Failed to make request for application: %s", source.ID, app.ID, uri.String(), err)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

		return
	}

	// yoink the xrhid from the parent request
	xrhid, ok := acr.c.Get(h.XRHID).(string)
	if !ok {
		acr.Logger().Warnf("couldn't pull xrhid from request")
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

		return
	}

	req.Header.Add(h.XRHID, xrhid)
	req.Header.Add(h.OrgID, source.Tenant.OrgID)
	req.Header.Add(h.AccountNumber, source.Tenant.ExternalTenant)
	req.Header.Add("Content-Type", "application/json;charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		acr.Logger().Errorf("[source_id: %d][application_id: %d] Error requesting availability status for application: %s", source.ID, app.ID, err)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginExternal)

		return
	}
	defer resp.Body.Close()

	// anything greater than 299 is bad, right??? right????
	if resp.StatusCode/100 > 2 {
		acr.Logger().Errorf("[source_id: %d][application_id: %d] Bad response from client: %d", source.ID, app.ID, resp.StatusCode)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginExternal)
	}

	acr.metricsService.IncrementSourcesAvailabilityCheckRequestsCounter()
}

type rhcConnectionStatusResponse struct {
	Status string `json:"status"`
}

// hit the RHC connector running in-cluster in order to check and see if the
// status for each RHC id is connected or disconnected
func (acr availabilityCheckRequester) RhcConnectionAvailabilityCheck(source *m.Source, headers []kafka.Header) {
	for i := range source.SourceRhcConnections {
		acr.pingRHC(source, &source.SourceRhcConnections[i].RhcConnection, headers)
	}
}

func (acr availabilityCheckRequester) pingRHC(source *m.Source, rhcConnection *m.RhcConnection, headers []kafka.Header) {
	if cloudConnectorUrl == "" {
		acr.Logger().Warnf("CLOUD_CONNECTOR_AVAILABILITY_CHECK_URL not set - skipping check for RHC Connection Availability Status [%v]", rhcConnection.RhcId)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

		return
	}

	if cloudConnectorStatusPath == "" {
		acr.Logger().Warnf("CLOUD_CONNECTOR_STATUS_PATH not set - skipping check for RHC Connection Availability Status [%v]", rhcConnection.RhcId)
		return
	}

	acr.Logger().Infof("Requesting Availability Check for RHC %v", rhcConnection.ID)

	// per: https://github.com/RedHatInsights/cloud-connector/blob/master/internal/controller/api/api.spec.json
	cloudConnectorStatusUrl := cloudConnectorUrl + "/" + rhcConnection.RhcId + cloudConnectorStatusPath

	// timeout after 10s
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", cloudConnectorStatusUrl, nil)
	if err != nil {
		acr.Logger().Warnf("Failed to create request for RHC Connection for ID %v, e: %v", source.ID, err)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginInternal)

		return
	}

	req.Header.Set("x-rh-cloud-connector-org-id", source.Tenant.OrgID)
	req.Header.Set("x-rh-cloud-connector-account", source.Tenant.ExternalTenant)
	req.Header.Set("x-rh-cloud-connector-client-id", cloudConnectorClientId)
	req.Header.Set("x-rh-cloud-connector-psk", cloudConnectorPsk)

	// Log the request before sending it.
	acr.Logger().Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] Created RHC connection status request`, source.ID, rhcConnection.ID, rhcConnection.RhcId)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		acr.Logger().Warnf("Failed to request connection_status for RHC ID [%v]: %v", rhcConnection.RhcId, err)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginExternal)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		acr.Logger().Warnf("Invalid return code received for RHC ID [%v]: %v", rhcConnection.RhcId, resp.StatusCode)
		b, _ := io.ReadAll(resp.Body)
		acr.Logger().Warnf("Body Returned from RHC ID [%v]: %s", rhcConnection.ID, b)
		acr.metricsService.IncrementSourcesAvailabilityCheckFailedRequestsCounter(metrics.OriginExternal)

		// updating status to unavailable
		acr.updateRhcStatus(source, "unavailable", unavailableRhc, rhcConnection, headers)

		return
	}

	// Increment the successes' counter, because even though we might face errors further down, we still successfully
	// sent the request and got a proper response.
	acr.metricsService.IncrementSourcesAvailabilityCheckRequestsCounter()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		acr.Logger().Warnf("failed to read body from request: %v", err)
		return
	}

	// Log everything from the response
	acr.Logger().Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status received response: %#v`, source.ID, rhcConnection.ID, rhcConnection.RhcId, resp)
	acr.Logger().Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status response status code: %d`, source.ID, rhcConnection.ID, rhcConnection.RhcId, resp.StatusCode)
	acr.Logger().Debugf(`[source_id: %d][rhc_connection_id: %d][rhc_connection_rhcid: %s] RHC connection status response body: %s`, source.ID, rhcConnection.ID, rhcConnection.RhcId, b)

	var status rhcConnectionStatusResponse

	err = json.Unmarshal(b, &status)
	if err != nil {
		acr.Logger().Warnf("failed to unmarshal response: %v", err)
		return
	}

	var (
		sanitizedStatus string
		errstr          string
	)

	switch status.Status {
	case "connected":
		sanitizedStatus = "available"
	case "disconnected":
		sanitizedStatus = "unavailable"
		errstr = disconnectedRhc
	default:
		acr.Logger().Warnf("Invalid status returned from RHC: %v", status.Status)
		return
	}

	acr.updateRhcStatus(source, sanitizedStatus, errstr, rhcConnection, headers)
}

func (acr availabilityCheckRequester) updateRhcStatus(source *m.Source, status string, errstr string, rhcConnection *m.RhcConnection, headers []kafka.Header) {
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

	requestParams, err := dao.NewRequestParamsFromContext(acr.c)
	if err != nil {
		acr.Logger().Warnf("failed to fetch request params from context: %v", err)
		return
	}

	err = dao.GetSourceDao(requestParams).Update(source)
	if err != nil {
		acr.Logger().Warnf("failed to update source availability status: %v", err)
		return
	}

	err = dao.GetRhcConnectionDao(requestParams).Update(rhcConnection)
	if err != nil {
		acr.Logger().Warnf("failed to update RHC Connection availability status: %v", err)
		return
	}

	l.Log.Debugf(`[source_id: %d][rhc_connection_id: %d] RHC Connection's status updated to "%s"`, source.ID, rhcConnection.ID, status)
	// we have to populate the Sources field in order to pass along the source_ids on the message.
	rhcConnection.Sources = []m.Source{*source}

	err = RaiseEvent("RhcConnection.update", rhcConnection, headers)
	if err != nil {
		acr.Logger().Warnf("error raising RhcConnection.update event: %v", err)
	}
}

func (acr availabilityCheckRequester) Logger() echo.Logger {
	return acr.c.Logger()
}

// isCostManagementApplication returns true when the given application is a Cost Management application.
func isCostManagementApplication(application m.Application) bool {
	return application.ApplicationType.Name == "/insights/platform/cost-management"
}
