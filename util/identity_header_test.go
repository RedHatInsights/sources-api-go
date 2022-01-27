package util

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func TestCreateIdentityHeader(t *testing.T) {
	out := XRhIdentityWithAccountNumber("1234")

	bytes, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Errorf("failed to decode generated x-rh-identity")
	}

	var identity identity.XRHID
	err = json.Unmarshal(bytes, &identity)
	if err != nil {
		t.Errorf("failed to unmarshal generated x-rh-identity")
	}

	if identity.Identity.AccountNumber != "1234" {
		t.Errorf("did not marshal correctly, got %v wanted %v", identity.Identity.AccountNumber, "1234")
	}
}
