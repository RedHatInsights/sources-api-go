package model

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

// setupEncryptionKeys installs a known primary key and (optionally) a legacy
// key, restoring whatever the process carried in afterwards.
func setupEncryptionKeys(t *testing.T, withLegacy bool) {
	t.Cleanup(func() { util.ResetEncryptionKeys() })

	util.OverrideEncryptionKey(strings.Repeat("P", 32))

	if withLegacy {
		util.OverrideLegacyEncryptionKey(strings.Repeat("L", 32))
	} else {
		util.ResetEncryptionKeys()
		util.OverrideEncryptionKey(strings.Repeat("P", 32))
	}
}

// TestGetPasswordPrefersGcmColumn verifies the read routing: when both
// columns are populated, the GCM column wins.
func TestGetPasswordPrefersGcmColumn(t *testing.T) {
	setupEncryptionKeys(t, true)

	auth := &Authentication{}

	if err := auth.SetPassword(util.StringRef("gcm-value")); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	// Simulate a row that also carries a stale legacy value: the GCM column
	// must still be the one that is read.
	staleLegacy, err := util.EncryptLegacyCBC("stale-legacy-value")
	if err != nil {
		t.Fatal(err)
	}
	auth.Password = util.StringRef(staleLegacy)

	pass, err := auth.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword failed: %v", err)
	}

	if pass == nil || *pass != "gcm-value" {
		t.Errorf("expected the GCM column to win, got %v", pass)
	}
}

// TestGetPasswordFallsBackToLegacyColumn verifies unmigrated rows are still
// readable through the legacy path.
func TestGetPasswordFallsBackToLegacyColumn(t *testing.T) {
	setupEncryptionKeys(t, true)

	legacy, err := util.EncryptLegacyCBC("legacy-value")
	if err != nil {
		t.Fatal(err)
	}

	auth := &Authentication{Password: util.StringRef(legacy)}

	pass, err := auth.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword failed: %v", err)
	}

	if pass == nil || *pass != "legacy-value" {
		t.Errorf("expected the legacy value, got %v", pass)
	}
}

// TestGetPasswordGcmFailureIsLoud verifies the no-silent-fallback rule: a GCM
// column value that fails to decrypt must produce an error even when a
// legacy value is sitting right next to it.
func TestGetPasswordGcmFailureIsLoud(t *testing.T) {
	setupEncryptionKeys(t, true)

	// GCM ciphertext encrypted with a key that is not the one its stored key
	// id names.
	util.OverrideEncryptionKey(strings.Repeat("X", 32))
	foreign, err := util.Encrypt("foreign-key-value")
	if err != nil {
		t.Fatal(err)
	}
	util.OverrideEncryptionKey(strings.Repeat("P", 32))

	legacy, err := util.EncryptLegacyCBC("legacy-value")
	if err != nil {
		t.Fatal(err)
	}

	auth := &Authentication{
		PasswordGcm:     util.StringRef(foreign),
		EncryptionKeyID: util.StringRef(util.ActiveKeyID()),
		Password:        util.StringRef(legacy),
	}

	if _, err := auth.GetPassword(); err == nil {
		t.Error("expected GetPassword to fail loudly on an undecryptable GCM value, not fall back to the legacy column")
	}
}

// TestGetPasswordRoutesByStoredKeyID covers the rotation scenario: the row
// was encrypted with a key that is no longer active, and the stored key id —
// not the active key — routes the decrypt.
func TestGetPasswordRoutesByStoredKeyID(t *testing.T) {
	t.Cleanup(func() { util.ResetEncryptionKeys() })

	ring := map[string]string{"a": strings.Repeat("A", 32), "b": strings.Repeat("B", 32)}

	// Write while "b" is active...
	util.OverrideEncryptionKeyring(ring, "b")

	auth := &Authentication{}
	if err := auth.SetPassword(util.StringRef("written-under-b")); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	if auth.EncryptionKeyID == nil || *auth.EncryptionKeyID != "b" {
		t.Fatalf("expected the row to be stamped with key id %q, got %v", "b", auth.EncryptionKeyID)
	}

	// ...then rotate the active key to "a" and read it back.
	util.OverrideEncryptionKeyring(ring, "a")

	pass, err := auth.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword after rotating the active key failed: %v", err)
	}

	if pass == nil || *pass != "written-under-b" {
		t.Errorf("expected the stored key id to route the decrypt, got %v", pass)
	}
}

// TestGetPasswordUnknownKeyID covers the misconfiguration case: a stored key
// id that is not in the keyring is a loud error, never a guess.
func TestGetPasswordUnknownKeyID(t *testing.T) {
	setupEncryptionKeys(t, false)

	encrypted, err := util.Encrypt("some-password")
	if err != nil {
		t.Fatal(err)
	}

	auth := &Authentication{
		PasswordGcm:     util.StringRef(encrypted),
		EncryptionKeyID: util.StringRef("no-such-key"),
	}

	if _, err := auth.GetPassword(); err == nil {
		t.Error("expected GetPassword to fail on an unknown encryption key id")
	}
}

// TestGetPasswordGcmWithoutKeyID covers the invariant violation: GCM
// ciphertext with no key id stamp must not be silently decrypted.
func TestGetPasswordGcmWithoutKeyID(t *testing.T) {
	setupEncryptionKeys(t, true)

	encrypted, err := util.Encrypt("some-password")
	if err != nil {
		t.Fatal(err)
	}

	auth := &Authentication{PasswordGcm: util.StringRef(encrypted)}

	if _, err := auth.GetPassword(); err == nil {
		t.Error("expected GetPassword to fail on a GCM password with no encryption key id")
	}
}

// TestGetPasswordLegacyColumnWithoutKey verifies the misconfiguration case is
// an explicit error, not a garbage "success".
func TestGetPasswordLegacyColumnWithoutKey(t *testing.T) {
	setupEncryptionKeys(t, false)

	auth := &Authentication{Password: util.StringRef("some-legacy-ciphertext")}

	_, err := auth.GetPassword()
	if err == nil {
		t.Error("expected an error when the legacy column is set but ENCRYPTION_KEY_OLD is not")
	}
}

func TestGetPasswordNilColumns(t *testing.T) {
	setupEncryptionKeys(t, true)

	auth := &Authentication{}

	pass, err := auth.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword failed: %v", err)
	}

	if pass != nil {
		t.Errorf("expected nil password, got %v", *pass)
	}
}

// TestSetPasswordDualWrite verifies the migration-window behavior: both
// columns are populated and both decrypt to the plaintext.
func TestSetPasswordDualWrite(t *testing.T) {
	setupEncryptionKeys(t, true)

	auth := &Authentication{}

	if err := auth.SetPassword(util.StringRef("dual-write-me")); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	if auth.PasswordGcm == nil {
		t.Fatal("expected the GCM column to be set")
	}

	if auth.Password == nil {
		t.Fatal("expected the legacy column to be dual-written while the migration window is open")
	}

	if auth.EncryptionKeyID == nil {
		t.Fatal("expected the row to be stamped with the active key id")
	}

	gcmPlain, err := util.DecryptGCMWithKeyID(*auth.PasswordGcm, *auth.EncryptionKeyID)
	if err != nil {
		t.Fatalf("DecryptGCMWithKeyID failed: %v", err)
	}

	if gcmPlain != "dual-write-me" {
		t.Errorf("GCM column: got %q, want %q", gcmPlain, "dual-write-me")
	}

	legacyPlain, err := util.DecryptLegacyCBC(*auth.Password)
	if err != nil {
		t.Fatalf("DecryptLegacyCBC failed: %v", err)
	}

	if legacyPlain != "dual-write-me" {
		t.Errorf("legacy column: got %q, want %q", legacyPlain, "dual-write-me")
	}
}

// TestSetPasswordSingleWrite verifies the post-contract behavior: with no
// legacy key configured, only the GCM column is written.
func TestSetPasswordSingleWrite(t *testing.T) {
	setupEncryptionKeys(t, false)

	auth := &Authentication{}

	if err := auth.SetPassword(util.StringRef("single-write-me")); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	if auth.PasswordGcm == nil {
		t.Fatal("expected the GCM column to be set")
	}

	if auth.Password != nil {
		t.Error("expected the legacy column to be untouched when no legacy key is configured")
	}
}

func TestSetPasswordNil(t *testing.T) {
	setupEncryptionKeys(t, true)

	auth := &Authentication{}

	if err := auth.SetPassword(nil); err != nil {
		t.Fatalf("SetPassword(nil) failed: %v", err)
	}

	if auth.PasswordGcm != nil || auth.Password != nil {
		t.Error("expected no password columns to be set for a nil password")
	}
}
