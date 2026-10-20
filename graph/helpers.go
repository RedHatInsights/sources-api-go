package graph

import (
	"context"

	m "github.com/RedHatInsights/sources-api-go/model"
)

// fetches the request data from the context
func getRequestDataFromCtx(ctx context.Context) *RequestData {
	r, ok := ctx.Value(RequestData{}).(*RequestData)
	if !ok {
		panic("could not pull tenant id from context")
	}

	return r
}

func userIdFromCtx(ctx context.Context) *int64 {
	return getRequestDataFromCtx(ctx).UserID
}

func tenantIdFromCtx(ctx context.Context) *int64 {
	return &getRequestDataFromCtx(ctx).TenantID
}

func sourceApplicationsFromCtx(ctx context.Context, id int64) []m.Application {
	mp := *getRequestDataFromCtx(ctx).applicationMap
	return mp[id]
}

func sourceEndpointsFromCtx(ctx context.Context, id int64) []m.Endpoint {
	mp := *getRequestDataFromCtx(ctx).endpointMap
	return mp[id]
}

func authenticationsFromCtx(ctx context.Context, resourceType string, id int64) []m.Authentication {
	mp := *getRequestDataFromCtx(ctx).authenticationMap
	out := make([]m.Authentication, 0)

	for _, auth := range mp[resourceType] {
		if id == auth.ResourceID {
			out = append(out, auth)
		}
	}

	return out
}

// sends the count into the requests channel, if the count wasn't requested we
// fetch it anyway since the DAO returns it
func sendCount(ctx context.Context, count int64) {
	getRequestDataFromCtx(ctx).CountChan <- int(count)
}

// gets the source count value from the ctx's channel
func getCount(ctx context.Context) int {
	return <-getRequestDataFromCtx(ctx).CountChan
}
