package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/rbac-client-go"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
)

// Client represents an RBAC client with which we can communicate with that service.
type Client interface {
	// Allowed checks whether the given principal in the "x-rh-identity" header has the "sources:*:*" permission or
	// not.
	Allowed(string) (bool, error)
	// GetDefaultWorkspace returns the default workspace for the given organization ID.
	GetDefaultWorkspace(orgId string) (string, error)
}

// rbacClientImpl is the internal implementation of the RBAC client.
type rbacClientImpl struct {
	// baseURL is the base URL of the RBAC service without the version suffix.
	baseURL string
	// legacyClient is the client that comes from the "rbac-client-go" dependency. There are two problems with that
	// legacy client:
	//
	// 1) The base URL needs to be specified with the version.
	// 2) It only supports the v1's "access" endpoint, but nothing else.
	//
	// We could probably replace it with an ad-hoc call from Sources, but we would need to implement the model and the
	// logic, and since the client already does it for us, it seems like a good idea to just keep it around for now.
	legacyClient rbac.Client
	rbacPSK      string
}

// NewRbacClient creates an RBAC client ready to be used by the callers.
func NewRbacClient(rbacHostname string, rbacPSK string) Client {
	// First set up the base URL for the client...
	client := rbacClientImpl{
		baseURL: fmt.Sprintf("%s/api/rbac", rbacHostname),
		rbacPSK: rbacPSK,
	}

	// ... and then create the legacy client with that base URL.
	client.legacyClient = rbac.NewClient(fmt.Sprintf("%s/v1", client.baseURL), "sources")

	return &client
}

func (r *rbacClientImpl) Allowed(xrhid string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	acl, err := r.legacyClient.GetAccess(ctx, xrhid, "")
	if err != nil {
		return false, err
	}

	return acl.IsAllowed("sources", "*", "*"), nil
}

func (r *rbacClientImpl) GetDefaultWorkspace(orgID string) (string, error) {
	//5 sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//Generate base url
	workspaceURL := fmt.Sprintf("%s/v2/workspaces/", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, workspaceURL, nil)
	if err != nil {
		logger.Log.WithField("org_id", orgID).Errorf("Failed to create request: %v", err)
		return "", fmt.Errorf("Failed to create request: %w", err)
	}

	//Add required Headers
	req.Header.Add("x-rh-rbac-psk", r.rbacPSK) //idk if this is what I am suposed to do or not

	q := req.URL.Query()
	q.Add("type", "default")
	q.Add("offset", "0")
	q.Add("limit", "2")
	req.URL.RawQuery = q.Encode()

	//Request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Log.WithField("org_id", orgID).Errorf("Failed to preform request: %v", err)
		return "", fmt.Errorf("Failed to preform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Log.WithField("org_id", orgID).Errorf("Bad response status: %d", resp.StatusCode)
		return "", fmt.Errorf("Bad response status: %d", resp.StatusCode)
	}

	var workspaceResp model.WorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&workspaceResp); err != nil {
		logger.Log.WithField("org_id", orgID).Errorf("Failed to decode response: %v", err)
		return "", fmt.Errorf("Failed to decode response: %w", err)
	}

	if len(workspaceResp.Data) != 1 {
		logger.Log.WithField("org_id", orgID).Warnf("More than one default workspace received from RBAC: %+v", workspaceResp.Data)
		return "", fmt.Errorf("Expected one default workspace but got %d", len(workspaceResp.Data))
	}

	workspaceID := workspaceResp.Data[0].ID
	logger.Log.WithField("org_id", orgID).Infof("Default workspace ID retrieved: %s", workspaceID)
	return workspaceID, nil
}
