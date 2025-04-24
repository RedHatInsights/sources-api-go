package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/RedHatInsights/rbac-client-go"
)

// Client represents an RBAC client with which we can communicate with that service.
type Client interface {
	// Allowed checks whether the given principal in the "x-rh-identity" header has the "sources:*:*" permission or
	// not.
	Allowed(string) (bool, error)
}

// rbacClientImpl is the internal implementation of the RBAC client.
type rbacClientImpl struct {
	// baseURL is the base URL of the RBAC service without the
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
}

// NewRbacClient creates an RBAC client ready to be used by the callers.
func NewRbacClient(rbacHostname string) Client {
	// First set up the base URL for the client...
	client := rbacClientImpl{
		baseURL: fmt.Sprintf("%s/api/rbac", rbacHostname),
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
