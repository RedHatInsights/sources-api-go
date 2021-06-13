package util

import (
	"fmt"
	"net/url"
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

func CollectionResponse(collection []interface{}, path string, count, limit, offset int) *Collection {
	first := fmt.Sprintf("limit=%d&offset=%d", limit, offset)
	last := fmt.Sprintf("limit=%d&offset=%d", limit, offset+limit)

	return &Collection{
		Data: collection,
		Meta: Metadata{
			Count:  count,
			Limit:  limit,
			Offset: offset,
		},
		Links: Links{
			First: fmt.Sprintf("%s/?%s", path, url.QueryEscape(first)),
			Last:  fmt.Sprintf("%s/?%s", path, url.QueryEscape(last)),
		},
	}
}
