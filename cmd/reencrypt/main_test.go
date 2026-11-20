package main

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

func setupBackfillKeys(t *testing.T) {
	t.Cleanup(func() { util.ResetEncryptionKeys() })

	util.OverrideEncryptionKey(strings.Repeat("P", 32))
	util.OverrideLegacyEncryptionKey(strings.Repeat("L", 32))
}

func TestTransformLegacyCiphertext(t *testing.T) {
	setupBackfillKeys(t)

	legacy, err := util.EncryptLegacyCBC("backfill-me")
	if err != nil {
		t.Fatalf("EncryptLegacyCBC failed: %v", err)
	}

	transformed, err := transformLegacyCiphertext(legacy)
	if err != nil {
		t.Fatalf("transformLegacyCiphertext failed: %v", err)
	}

	// The result must be readable as GCM with the active key — that is the
	// whole point of the bootstrap.
	plaintext, err := util.DecryptGCM(transformed)
	if err != nil {
		t.Fatalf("DecryptGCM on transformed value failed: %v", err)
	}

	if plaintext != "backfill-me" {
		t.Errorf("got %q, want %q", plaintext, "backfill-me")
	}

	// And it must NOT be readable through the legacy path with the old key:
	// the value moved columns, so it must have moved formats.
	if _, err := util.DecryptLegacyCBC(transformed); err == nil {
		t.Error("transformed value unexpectedly decrypted through the legacy path")
	}
}

func TestTransformLegacyCiphertextRejectsGarbage(t *testing.T) {
	setupBackfillKeys(t)

	if _, err := transformLegacyCiphertext("not-a-valid-ciphertext"); err == nil {
		t.Error("expected an error for undecryptable legacy input")
	}
}

// TestRotateGCMCiphertext covers the future-rotation transform: decrypt with
// the recorded key id, re-encrypt with the active key.
func TestRotateGCMCiphertext(t *testing.T) {
	t.Cleanup(func() { util.ResetEncryptionKeys() })

	ring := map[string]string{"old": strings.Repeat("O", 32), "new": strings.Repeat("N", 32)}

	// The row was written while "old" was active.
	util.OverrideEncryptionKeyring(ring, "old")

	ciphertext, err := util.Encrypt("rotate-me")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	oldID := "old"

	// The rotation runs with "new" active.
	util.OverrideEncryptionKeyring(ring, "new")

	rotated, err := rotateGCMCiphertext(ciphertext, &oldID)
	if err != nil {
		t.Fatalf("rotateGCMCiphertext failed: %v", err)
	}

	plaintext, err := util.DecryptGCMWithKeyID(rotated, "new")
	if err != nil {
		t.Fatalf("DecryptGCMWithKeyID on rotated value failed: %v", err)
	}

	if plaintext != "rotate-me" {
		t.Errorf("got %q, want %q", plaintext, "rotate-me")
	}

	if rotated == ciphertext {
		t.Error("expected the rotated ciphertext to differ from the original")
	}
}

// TestRotateGCMCiphertextNilID covers the heal case: rows written before key
// stamping carry no id and are assumed to use the then-active key.
func TestRotateGCMCiphertextNilID(t *testing.T) {
	setupBackfillKeys(t)

	ciphertext, err := util.Encrypt("unstamped-row")
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := rotateGCMCiphertext(ciphertext, nil)
	if err != nil {
		t.Fatalf("rotateGCMCiphertext with nil key id failed: %v", err)
	}

	plaintext, err := util.DecryptGCM(rotated)
	if err != nil {
		t.Fatalf("DecryptGCM on rotated value failed: %v", err)
	}

	if plaintext != "unstamped-row" {
		t.Errorf("got %q, want %q", plaintext, "unstamped-row")
	}
}

func TestRotateGCMCiphertextUnknownID(t *testing.T) {
	setupBackfillKeys(t)

	ciphertext, err := util.Encrypt("some-row")
	if err != nil {
		t.Fatal(err)
	}

	badID := "no-such-key"

	if _, err := rotateGCMCiphertext(ciphertext, &badID); err == nil {
		t.Error("expected an error for a key id that is not in the keyring")
	}
}

// TestMigrateRowDryRunBootstrap covers the dry-run path without touching the
// database: valid rows report as would-migrate, undecryptable rows fail, and
// a nil legacy password fails defensively.
func TestMigrateRowDryRunBootstrap(t *testing.T) {
	setupBackfillKeys(t)

	legacy, err := util.EncryptLegacyCBC("row-password")
	if err != nil {
		t.Fatal(err)
	}

	row := pendingAuth{ID: 42, PasswordHash: &legacy}

	result := migrateRow(nil, row, false, true)
	if result.status != statusMigrated {
		t.Errorf("expected dry-run on a valid row to report migrated, got status %d (%v)", result.status, result.err)
	}

	badRow := pendingAuth{ID: 43, PasswordHash: util.StringRef("garbage")}

	result = migrateRow(nil, badRow, false, true)
	if result.status != statusFailed {
		t.Errorf("expected dry-run on an undecryptable row to report failed, got status %d", result.status)
	}

	if result.err == nil {
		t.Error("expected an error on the failed row")
	}

	nilRow := pendingAuth{ID: 44}

	result = migrateRow(nil, nilRow, false, true)
	if result.status != statusFailed {
		t.Errorf("expected a nil legacy password to report failed, got status %d", result.status)
	}
}

// TestMigrateRowDryRunRotate covers the rotate mode's dry-run path: rows
// stamped with an old key id report as would-migrate, and rows without a GCM
// password fail.
func TestMigrateRowDryRunRotate(t *testing.T) {
	t.Cleanup(func() { util.ResetEncryptionKeys() })

	ring := map[string]string{"old": strings.Repeat("O", 32), "new": strings.Repeat("N", 32)}

	util.OverrideEncryptionKeyring(ring, "old")

	ciphertext, err := util.Encrypt("old-key-row")
	if err != nil {
		t.Fatal(err)
	}

	oldID := "old"

	util.OverrideEncryptionKeyring(ring, "new")

	row := pendingAuth{ID: 45, PasswordHashGcm: &ciphertext, EncryptionKeyID: &oldID}

	result := migrateRow(nil, row, true, true)
	if result.status != statusMigrated {
		t.Errorf("expected dry-run on a rotatable row to report migrated, got status %d (%v)", result.status, result.err)
	}

	noGcmRow := pendingAuth{ID: 46, PasswordHash: util.StringRef("legacy-value")}

	result = migrateRow(nil, noGcmRow, true, true)
	if result.status != statusFailed {
		t.Errorf("expected a row without a GCM password to report failed in rotate mode, got status %d", result.status)
	}
}

// TestMigrateRowStatusValues pins the status ordering that the summary
// counters in main depend on.
func TestMigrateRowStatusValues(t *testing.T) {
	if statusMigrated == statusSkipped || statusSkipped == statusFailed || statusMigrated == statusFailed {
		t.Error("row status constants must be distinct")
	}
}
