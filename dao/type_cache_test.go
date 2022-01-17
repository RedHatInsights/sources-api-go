package dao

import "testing"

func TestStaticCache(t *testing.T) {
	Static = typeCache{
		sourceTypes:      map[string]int64{"amazon": 17},
		applicationTypes: map[string]int64{"cloud-meter": 1},
	}

	if Static.GetApplicationTypeId("cloud-meter") != 1 {
		t.Errorf("Incorrect ApplicationType ID returned")
	}

	if Static.GetSourceTypeId("amazon") != 17 {
		t.Errorf("Incorrect SourceType ID returned")
	}
}
