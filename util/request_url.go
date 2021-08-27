package util

import (
	"fmt"
	m "github.com/RedHatInsights/sources-api-go/model"
	"strconv"
	"strings"
)

type RequestURL struct {
	ParamID string
	ParsedID int64
	Path string
	PrimaryCollection string
	SecondaryCollection string
}

func NewRequestURL(path string, paramID string) (*RequestURL, error) {
	request := &RequestURL{Path: path, ParamID: paramID}
	err := request.parse()
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (request *RequestURL) parse() error {
	pathWithoutPrefix := strings.Replace(request.Path, "/api/sources/v3.1/", "", 1)
	pathParts := strings.Split(pathWithoutPrefix, "/")
	switch len(pathParts) {
	case 1, 2:
		request.PrimaryCollection = pathParts[0]
	case 3:
		request.PrimaryCollection = pathParts[0]
		request.SecondaryCollection = pathParts[2]
	default:
		return fmt.Errorf("failed to parse url: %v", request.Path)
	}

	if request.ParamID != "" {
		id, err := strconv.ParseInt(request.ParamID, 10, 64)
		if err != nil {
			return err
		}
		request.ParsedID = id
	}

	return nil
}

func (request *RequestURL) IsSubCollection() bool {
	return request.SecondaryCollection != ""
}

func (request *RequestURL) PrimaryResource() interface{} {
	switch request.PrimaryCollection {
	case "application_types":
		return m.ApplicationType{Id: request.ParsedID}
	case "sources":
		return m.Source{ID: request.ParsedID}
	case "source_types":
		return m.SourceType{Id: request.ParsedID}
	default:
		panic("Collection " + request.PrimaryCollection + " not found.")
	}
}
