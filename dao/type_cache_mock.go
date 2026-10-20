package dao

import "github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"

func PopulateMockStaticTypeCache() error {
	tc := typeCache{}
	tc.sourceTypes = make(map[string]int64)
	tc.applicationTypes = make(map[string]int64)

	for _, st := range fixtures.TestSourceTypeData {
		tc.sourceTypes[st.Name] = st.Id
	}

	for _, at := range fixtures.TestApplicationTypeData {
		tc.applicationTypes[at.Name] = at.Id
	}

	Static = tc

	return nil
}
