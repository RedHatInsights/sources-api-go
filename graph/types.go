package graph

/*
	Struct to track any information on the current GraphQL request

	Currently using to just track the tenantID as well as the GUID key to set
	the count in redis (so we can use it for the metadata later if needed)
*/
type RequestData struct {
	TenantID  int64
	CountChan chan int
}
