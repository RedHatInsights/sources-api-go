package graph

import (
	"context"
)

// fetches the request data from the context
func getRequestData(ctx context.Context) *RequestData {
	r, ok := ctx.Value(RequestData{}).(*RequestData)
	if !ok {
		panic("could not pull tenant id from context")
	}

	return r
}

func tenantIdFromContext(ctx context.Context) *int64 {
	return &getRequestData(ctx).TenantID
}

// sends the count into the requests channel, if the count wasn't requested we
// fetch it anyway since the DAO returns it
func sendCount(ctx context.Context, count int64) {
	getRequestData(ctx).CountChan <- int(count)
}

// gets the source count value from the ctx's channel
func getCount(ctx context.Context) int {
	return <-getRequestData(ctx).CountChan
}
