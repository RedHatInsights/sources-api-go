package dao

import (
	"strings"

	m "github.com/RedHatInsights/sources-api-go/model"
)

// Package level Type cache - to be accessed from anywhere
var Static typeCache

/*
typeCache is a small struct that just contains the map between
application/source types and their keys in the database.

The cache is populated at startup so one can fetch foreign keys when
building objects easily.
*/
type typeCache struct {
	sourceTypes      map[string]int64
	applicationTypes map[string]int64
}

// returns the source type id for a given name, 0 if not found.
func (tc *typeCache) GetSourceTypeId(name string) int64 {
	return tc.sourceTypes[name]
}

// inverse of the above function - takes an id and returns the string name for
// the source type id
func (tc *typeCache) GetSourceTypeName(id int64) string {
	for k, v := range tc.sourceTypes {
		if v == id {
			return k
		}
	}

	return ""
}

/*
	    Returns the application type id for a given name, 0 if not found.

		Can search via name or display name for application types, there is also a
		shortcut for the last place in the "path" of the name e.g. `/this/is/a/type`
		is also available as `type`
*/
func (tc *typeCache) GetApplicationTypeId(name string) int64 {
	return tc.applicationTypes[name]
}

// inverse of id function - takes an id and returns the short-form name of the
// application type
func (tc *typeCache) GetApplicationTypeName(id int64) string {
	for k, v := range tc.applicationTypes {
		if v == id && !strings.Contains(k, "/") {
			return k
		}
	}

	return ""
}

// same as above function - but fetches the long-form name
func (tc *typeCache) GetApplicationTypeFullName(id int64) string {
	for k, v := range tc.applicationTypes {
		if v == id && strings.Contains(k, "/") {
			return k
		}
	}

	return ""
}

/*
Fetches every Source+Application Type record from the database and builds
out the cache. Returns an error if it fails (e.g. the app shouldn't be
running then)
*/
func PopulateStaticTypeCache() error {
	tc := typeCache{}
	tc.sourceTypes = make(map[string]int64)
	tc.applicationTypes = make(map[string]int64)

	err := populateSourceTypes(tc)
	if err != nil {
		return err
	}
	err = populateApplicationTypes(tc)
	if err != nil {
		return err
	}

	Static = tc
	return nil
}

func populateSourceTypes(cache typeCache) error {
	sourceTypes := make([]m.SourceType, 0)
	result := DB.Model(&m.SourceType{}).Scan(&sourceTypes)
	if result.Error != nil {
		return result.Error
	}

	for _, st := range sourceTypes {
		cache.sourceTypes[st.Name] = st.Id
	}

	return nil
}

func populateApplicationTypes(cache typeCache) error {
	appTypes := make([]m.ApplicationType, 0)
	result := DB.Model(&m.ApplicationType{}).Scan(&appTypes)
	if result.Error != nil {
		return result.Error
	}

	for _, at := range appTypes {
		// The entire path name
		cache.applicationTypes[at.Name] = at.Id

		// the last part of the path name (usually used in bulk_create)
		parts := strings.Split(at.Name, "/")
		shortName := parts[len(parts)-1]
		cache.applicationTypes[shortName] = at.Id
	}

	return nil
}
