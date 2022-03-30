package graph

/*
	Struct to track any information on the current GraphQL request

	Currently only storing the TenantID of the current request and
	a channel which we can send the count from the main goroutine to
	the one that is handling the metadata querying
*/
type RequestData struct {
	TenantID  int64
	CountChan chan int
}
