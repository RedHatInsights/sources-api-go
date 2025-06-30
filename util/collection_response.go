package util

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type Collection struct {
	Data  []interface{} `json:"data"`
	Meta  Metadata      `json:"meta"`
	Links Links         `json:"links"`
}

type Metadata struct {
	Count  int `json:"count"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

func CollectionResponse(collection []interface{}, req *http.Request, count, limit, offset int) *Collection {
	var first, last string

	q := req.URL.Query()

	// set the "first" link with same limit+offset (what they requested)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	params, _ := url.PathUnescape(q.Encode())
	first = fmt.Sprintf("%v?%v", req.URL.Path, params)

	// set the "last" link with limit+offset set for the next page
	q.Set("offset", strconv.Itoa(offset+limit))
	params, _ = url.PathUnescape(q.Encode())
	last = fmt.Sprintf("%v?%v", req.URL.Path, params)

	// set offset based on limit size aka page size
	links := Links{
		First: first,
		Last:  last,
	}

	return &Collection{
		Data: collection,
		Meta: Metadata{
			Count:  count,
			Limit:  limit,
			Offset: offset,
		},
		Links: links,
	}
}
