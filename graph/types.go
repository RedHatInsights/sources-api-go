package graph

import (
	"sync"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const defaultLimit = 500

// Struct to track any information on the current GraphQL request
type RequestData struct {
	TenantID  int64
	CountChan chan int

	// Mutex + pointer to a collection so that we only load applications or
	// endpoints _one time_ from the database.
	//
	// this way we only make one query (when requested) instead of N+1
	ApplicationMutex    *sync.Mutex
	applicationMap      *map[int64][]m.Application
	EndpointMutex       *sync.Mutex
	endpointMap         *map[int64][]m.Endpoint
	AuthenticationMutex *sync.Mutex
	authenticationMap   *map[string][]m.Authentication
	SourceMutex         *sync.Mutex
	sourceIdList        *[]string
}

// wrapper around a mutex that only loads applications up one time and makes any
// other routines wait until the map is ready
func (rd *RequestData) EnsureApplicationsAreLoaded() error {
	if rd.applicationMap != nil {
		return nil
	}

	rd.ApplicationMutex.Lock()
	defer rd.ApplicationMutex.Unlock()

	// waiting until the sourceIDs we need are loaded
	rd.SourceMutex.Lock()
	defer rd.SourceMutex.Unlock()

	// load up the application map if it isn't present - this will only happen
	// once and any other threads will wait for it to complete. We have to check
	// again due to the fact that multiple threads might have locked this the
	// first time
	if rd.applicationMap == nil {
		apps, _, err := dao.GetApplicationDao(&rd.TenantID).List(defaultLimit, 0, []util.Filter{{Name: "source_id", Value: *rd.sourceIdList}})
		if err != nil {
			return err
		}

		mp := make(map[int64][]m.Application)
		for _, app := range apps {
			mp[app.SourceID] = append(mp[app.SourceID], app)
		}

		// persist the map on the context - so it can be used by all subsequent routines
		rd.applicationMap = &mp
	}

	return nil
}

// largely a copy/paste of above - so I won't duplicate the comments
func (rd *RequestData) EnsureEndpointsAreLoaded() error {
	if rd.endpointMap != nil {
		return nil
	}

	rd.EndpointMutex.Lock()
	defer rd.EndpointMutex.Unlock()

	// waiting until the sourceIDs we need are loaded
	rd.SourceMutex.Lock()
	defer rd.SourceMutex.Unlock()

	if rd.endpointMap == nil {
		endpts, _, err := dao.GetEndpointDao(&rd.TenantID).List(defaultLimit, 0, []util.Filter{{Name: "source_id", Value: *rd.sourceIdList}})
		if err != nil {
			return err
		}

		mp := make(map[int64][]m.Endpoint)
		for _, endpt := range endpts {
			mp[endpt.SourceID] = append(mp[endpt.SourceID], endpt)
		}

		rd.endpointMap = &mp
	}

	return nil
}

// largely a copy/paste of above - so I won't duplicate the comments
func (rd *RequestData) EnsureAuthenticationsAreLoaded() error {
	if rd.authenticationMap != nil {
		return nil
	}

	rd.AuthenticationMutex.Lock()
	defer rd.AuthenticationMutex.Unlock()

	// waiting until the sourceIDs we need are loaded
	rd.SourceMutex.Lock()
	defer rd.SourceMutex.Unlock()

	if rd.authenticationMap == nil {
		auths, _, err := dao.GetAuthenticationDao(&rd.TenantID).List(defaultLimit, 0, []util.Filter{{Name: "source_id", Value: *rd.sourceIdList}})
		if err != nil {
			return err
		}

		mp := make(map[string][]m.Authentication)
		for _, auth := range auths {
			mp[auth.ResourceType] = append(mp[auth.ResourceType], auth)
		}

		rd.authenticationMap = &mp
	}

	return nil
}

// effectively just sets the source ID list and then unlocks the mutext so the
// rest of the subresources can run
func (rd *RequestData) SetSourceIDs(ids []string) {
	rd.sourceIdList = &ids
	rd.SourceMutex.Unlock()
}
