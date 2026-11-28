package util

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

// testKeyA is the historical test key: the legacy CBC vectors below were
// produced with it.
const testKeyAB64 = "mD0I8u2luw52GIQpEteYWQu2UxsWP4kacSBhjgAh5C9"

// setPrimaryTestKey loads testKeyA as the primary (GCM) key.
func setPrimaryTestKey() {
	bin, _ := base64.RawStdEncoding.DecodeString(testKeyAB64)
	OverrideEncryptionKey(string(bin))
}

// setLegacyTestKey loads a key distinct from the primary as the pre-rotation
// (legacy CBC) key.
func setLegacyTestKey() {
	OverrideLegacyEncryptionKey(strings.Repeat("L", 32))
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setPrimaryTestKey()

	passwords := []string{
		"sources-api-tests",
		"short",
		"a-very-long-password-that-exceeds-multiple-block-sizes-1234567890",
		"",
		"special chars: !@#$%^&*()",
		strings.Repeat("x", 256),
	}

	for _, pw := range passwords {
		encrypted, err := Encrypt(pw)
		if err != nil {
			t.Fatalf("Encrypt(%q) failed: %v", pw, err)
		}

		decrypted, err := DecryptGCM(encrypted)
		if err != nil {
			t.Fatalf("DecryptGCM(%q) failed: %v", pw, err)
		}

		if decrypted != pw {
			t.Errorf("round-trip failed for %q: got %q", pw, decrypted)
		}
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	setPrimaryTestKey()

	enc1, err := Encrypt("same-password")
	if err != nil {
		t.Fatal(err)
	}

	enc2, err := Encrypt("same-password")
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Error("two encryptions of the same plaintext produced identical ciphertext — nonce is not random")
	}
}

func TestEncryptOutputHasGCMScheme(t *testing.T) {
	setPrimaryTestKey()

	enc, err := Encrypt("test")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}

	if len(raw) == 0 || raw[0] != schemeGCM {
		t.Errorf("encrypted output should start with GCM scheme byte 0x%02x, got 0x%02x", schemeGCM, raw[0])
	}
}

// legacyCBCTestVector is "sources-api-tests" encrypted with the old CBC
// zero-IV scheme and testKeyA.
const legacyCBCTestVector = "f80zhczL8GTPqCdlU8WX7m+8BgCZgwXERNGYUF7J+lU"

func TestDecryptLegacyCBCWithOldKey(t *testing.T) {
	// The rotation scenario: the primary key is new, the legacy vector was
	// encrypted with the pre-rotation key (here: testKeyA loaded as legacy).
	ResetEncryptionKeys()
	bin, _ := base64.RawStdEncoding.DecodeString(testKeyAB64)
	OverrideEncryptionKey(strings.Repeat("P", 32))
	OverrideLegacyEncryptionKey(string(bin))

	out, err := DecryptLegacyCBC(legacyCBCTestVector)
	if err != nil {
		t.Fatalf("DecryptLegacyCBC failed: %v", err)
	}

	if out != "sources-api-tests" {
		t.Errorf("legacy CBC decryption: got %q, want %q", out, "sources-api-tests")
	}
}

func TestDecryptLegacyCBCWithoutRotation(t *testing.T) {
	// The cipher-only scenario: no ENCRYPTION_KEY_OLD, so the sniffing
	// Decrypt falls back to the primary key for legacy ciphertext.
	setPrimaryTestKey()

	out, err := Decrypt(legacyCBCTestVector)
	if err != nil {
		t.Fatalf("legacy CBC Decrypt failed: %v", err)
	}

	if out != "sources-api-tests" {
		t.Errorf("legacy CBC decryption: got %q, want %q", out, "sources-api-tests")
	}
}

// TestDecryptCBCCollisionWithSchemeByte verifies that a legacy CBC-encrypted
// password whose raw ciphertext happens to start with byte 0x01 (the GCM
// scheme marker) is still decryptable via the sniffing Decrypt: it must try
// GCM first and, on authentication failure, fall back to the CBC decoder.
func TestDecryptCBCCollisionWithSchemeByte(t *testing.T) {
	setPrimaryTestKey()

	// "000428-cbc-test" encrypted with the old CBC zero-IV scheme and testKeyA
	// produces a ciphertext whose first decoded byte is 0x01 — the same value
	// as schemeGCM. Without the GCM-fallback logic this would be misrouted to
	// decodeGCM and fail.
	legacy := "AUPz0DZXAcXSOu0/R8DqtA"

	raw, err := base64.RawStdEncoding.DecodeString(legacy)
	if err != nil {
		t.Fatal(err)
	}

	if raw[0] != schemeGCM {
		t.Fatalf("test vector first byte is 0x%02x, expected 0x%02x", raw[0], schemeGCM)
	}

	decrypted, err := Decrypt(legacy)
	if err != nil {
		t.Fatalf("Decrypt of CBC ciphertext starting with scheme byte 0x01 should succeed via fallback, got: %v", err)
	}

	want := "000428-cbc-test"
	if decrypted != want {
		t.Errorf("expected %q, got %q", want, decrypted)
	}
}

// TestGCMWrongKeyFailsLoudly is the test the old dual-key fallback needed:
// GCM ciphertext encrypted with a different key must produce an error, never
// garbage plaintext. This is the property that makes wrong-key detection
// possible (and that AES-CBC lacked).
func TestGCMWrongKeyFailsLoudly(t *testing.T) {
	setPrimaryTestKey()

	encrypted, err := Encrypt("rotate-me")
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the "wrong" side of a rotation: a different key.
	OverrideEncryptionKey(strings.Repeat("Q", 32))

	if _, err := DecryptGCM(encrypted); err == nil {
		t.Error("expected DecryptGCM to fail on ciphertext encrypted with a different key")
	}
}

// TestDecryptGCMDoesNotSilentlyFallBackToLegacy verifies the routing rule
// production code relies on: GCM values fail loudly, they never fall through
// to the legacy decoder.
func TestDecryptGCMDoesNotSilentlyFallBackToLegacy(t *testing.T) {
	ResetEncryptionKeys()
	setPrimaryTestKey()
	setLegacyTestKey()

	// A legacy ciphertext is not valid GCM input: it must error, not return
	// garbage.
	if _, err := DecryptGCM(legacyCBCTestVector); err == nil {
		t.Error("expected DecryptGCM to reject legacy CBC ciphertext")
	}
}

func TestLegacyFunctionsRequireLegacyKey(t *testing.T) {
	ResetEncryptionKeys()
	setPrimaryTestKey()

	if _, err := EncryptLegacyCBC("x"); err == nil {
		t.Error("expected EncryptLegacyCBC to fail without a legacy key")
	}

	if _, err := DecryptLegacyCBC(legacyCBCTestVector); err == nil {
		t.Error("expected DecryptLegacyCBC to fail without a legacy key")
	}
}

// TestLegacyRoundTripDualKey exercises the full rotation data flow: legacy
// column values are written with the old key, read back with it, and are
// never misread by the GCM path.
func TestLegacyRoundTripDualKey(t *testing.T) {
	ResetEncryptionKeys()
	setPrimaryTestKey()
	setLegacyTestKey()

	if !LegacyKeyPresent() {
		t.Fatal("expected LegacyKeyPresent() after OverrideLegacyEncryptionKey")
	}

	legacy, err := EncryptLegacyCBC("pre-rotation-secret")
	if err != nil {
		t.Fatalf("EncryptLegacyCBC failed: %v", err)
	}

	out, err := DecryptLegacyCBC(legacy)
	if err != nil {
		t.Fatalf("DecryptLegacyCBC failed: %v", err)
	}

	if out != "pre-rotation-secret" {
		t.Errorf("legacy round-trip: got %q, want %q", out, "pre-rotation-secret")
	}

	// The GCM path must not "successfully" decrypt legacy data.
	if _, err := DecryptGCM(legacy); err == nil {
		t.Error("expected DecryptGCM to fail on legacy CBC ciphertext")
	}

	// The sniffing Decrypt handles both formats.
	gcm, err := Encrypt("post-rotation-secret")
	if err != nil {
		t.Fatal(err)
	}

	sniffed, err := Decrypt(gcm)
	if err != nil {
		t.Fatalf("Decrypt of GCM ciphertext failed: %v", err)
	}

	if sniffed != "post-rotation-secret" {
		t.Errorf("sniffing Decrypt: got %q, want %q", sniffed, "post-rotation-secret")
	}
}

func TestDecryptTamperedGCM(t *testing.T) {
	setPrimaryTestKey()

	enc, err := Encrypt("test-tamper")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}

	// flip a byte in the ciphertext
	raw[len(raw)-1] ^= 0xff
	tampered := base64.RawStdEncoding.EncodeToString(raw)

	if _, err := DecryptGCM(tampered); err == nil {
		t.Error("expected GCM decryption to fail on tampered ciphertext")
	}
}

func TestBadFormat(t *testing.T) {
	setPrimaryTestKey()

	_, err := Decrypt("a bad thing")
	if err == nil {
		t.Error("expected an error for bad base64 input")
	}

	_, err = DecryptGCM("this is not a real string")
	if err == nil {
		t.Error("expected an error for invalid ciphertext")
	}
}

func TestNoKey(t *testing.T) {
	ResetEncryptionKeys()

	_, err := Decrypt("another thing")
	if err == nil {
		t.Error("expected an error with no key")
	}

	if err.Error() != "no encryption key present" {
		t.Errorf("bad error message: %v", err.Error())
	}

	_, err = Encrypt("test")
	if err == nil {
		t.Error("expected an error with no key")
	}
}

func TestResetEncryptionKeys(t *testing.T) {
	setPrimaryTestKey()
	setLegacyTestKey()

	ResetEncryptionKeys()

	if LegacyKeyPresent() {
		t.Error("expected LegacyKeyPresent() to be false after ResetEncryptionKeys")
	}
}

func TestInitializeEncryptionLoadsLegacyKey(t *testing.T) {
	t.Setenv("ENCRYPTION_KEYS", "")
	t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", "")
	t.Setenv("ENCRYPTION_KEY", testKeyAB64)
	t.Setenv("ENCRYPTION_KEY_OLD", base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("L", 32))))

	InitializeEncryption()

	if !LegacyKeyPresent() {
		t.Error("expected the legacy key to be loaded from ENCRYPTION_KEY_OLD")
	}

	// The self-test ran as part of initialization; verify both paths work.
	if _, err := Encrypt("probe"); err != nil {
		t.Errorf("Encrypt after InitializeEncryption failed: %v", err)
	}

	if _, err := EncryptLegacyCBC("probe"); err != nil {
		t.Errorf("EncryptLegacyCBC after InitializeEncryption failed: %v", err)
	}
}

func TestInitializeEncryptionPanicsOnBadLegacyKey(t *testing.T) {
	t.Setenv("ENCRYPTION_KEYS", "")
	t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", "")
	t.Setenv("ENCRYPTION_KEY", testKeyAB64)
	t.Setenv("ENCRYPTION_KEY_OLD", "!!!not-base64!!!")

	defer func() {
		if recover() == nil {
			t.Error("expected InitializeEncryption to panic on an invalid ENCRYPTION_KEY_OLD")
		}
	}()

	InitializeEncryption()
}

func TestSetDefaultEncryptionKey(t *testing.T) {
	t.Setenv("ENCRYPTION_KEYS", "")
	t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", "")
	t.Setenv("ENCRYPTION_KEY", "")
	t.Setenv("ENCRYPTION_KEY_OLD", "")

	setErr := os.Setenv("ENCRYPTION_KEY", "")
	if setErr != nil {
		t.Errorf("encryption key unable to set as empty string: %v", setErr)
	}

	InitializeEncryption()

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey != "YWFhYWFhYWFhYWFhYWFhYQ" {
		t.Errorf("Wrong encryption key! setDefaultEncryptionKey() did not work properly")
	}
}

func TestEncryptDecryptWithShortKey(t *testing.T) {
	// 16-byte key (AES-128) — matches the dev encryption key
	OverrideEncryptionKey("aaaaaaaaaaaaaaaa")

	pw := "test-with-short-key"

	enc, err := Encrypt(pw)
	if err != nil {
		t.Fatalf("Encrypt with 16-byte key failed: %v", err)
	}

	dec, err := DecryptGCM(enc)
	if err != nil {
		t.Fatalf("DecryptGCM with 16-byte key failed: %v", err)
	}

	if dec != pw {
		t.Errorf("round-trip with 16-byte key: got %q, want %q", dec, pw)
	}
}

func TestPadding(t *testing.T) {
	out := padString("thing", 16)

	if len(out) != 16 {
		t.Errorf("padding was not added properly to string")
	}
}

func TestPaddingNegativeBlockSize(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("execution should have panicked but did not")
		}
	}()

	padString("boom", -1)
}

// keyringTestKeys builds an ENCRYPTION_KEYS value from two distinct raw
// keys, base64-encoded the way the environment expects.
func keyringTestKeys(idA, keyA, idB, keyB string) string {
	return idA + ":" + base64.RawStdEncoding.EncodeToString([]byte(keyA)) +
		"," + idB + ":" + base64.RawStdEncoding.EncodeToString([]byte(keyB))
}

// TestEncryptionKeyringMapRoundTrip covers the full map configuration: two
// keys, one active, decryption routed by the stored key id.
func TestEncryptionKeyringMapRoundTrip(t *testing.T) {
	t.Setenv("ENCRYPTION_KEYS", keyringTestKeys("k1", strings.Repeat("A", 32), "k2", strings.Repeat("B", 32)))
	t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", "k2")

	InitializeEncryption()

	if ActiveKeyID() != "k2" {
		t.Errorf("expected active key id %q, got %q", "k2", ActiveKeyID())
	}

	encrypted, err := Encrypt("keyring-secret")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// The ciphertext is readable through the id it was stamped with...
	decrypted, err := DecryptGCMWithKeyID(encrypted, "k2")
	if err != nil {
		t.Fatalf("DecryptGCMWithKeyID with the active id failed: %v", err)
	}

	if decrypted != "keyring-secret" {
		t.Errorf("got %q, want %q", decrypted, "keyring-secret")
	}

	// ...and NOT through the other keyring entry.
	if _, err := DecryptGCMWithKeyID(encrypted, "k1"); err == nil {
		t.Error("expected decryption with the wrong key id to fail")
	}
}

// TestDecryptGCMWithKeyIDUnknownID pins the loud-failure rule: an id that is
// not in the keyring is an error naming the id, never a guess.
func TestDecryptGCMWithKeyIDUnknownID(t *testing.T) {
	setPrimaryTestKey()

	encrypted, err := Encrypt("probe")
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptGCMWithKeyID(encrypted, "no-such-key")
	if err == nil {
		t.Fatal("expected an error for an unknown key id")
	}

	if !strings.Contains(err.Error(), "no-such-key") {
		t.Errorf("error should name the unknown key id, got: %v", err)
	}
}

func TestOverrideEncryptionKeyUsesDefaultID(t *testing.T) {
	OverrideEncryptionKey(strings.Repeat("P", 32))

	if ActiveKeyID() != "v1" {
		t.Errorf("expected the shorthand override to use the default id %q, got %q", "v1", ActiveKeyID())
	}
}

// TestInitializeEncryptionPanicsOnBadKeyring covers every configuration
// error that must stop the process at boot rather than at request time.
func TestInitializeEncryptionPanicsOnBadKeyring(t *testing.T) {
	validA := base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("A", 32)))
	validB := base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("B", 32)))

	cases := map[string]struct {
		keys   string
		active string
	}{
		"duplicate key ids":       {"k1:" + validA + ",k1:" + validB, ""},
		"active id not in map":    {"k1:" + validA, "k9"},
		"active id required":      {"k1:" + validA + ",k2:" + validB, ""},
		"entry missing separator": {"k1", "k1"},
		"entry bad base64":        {"k1:!!!not-base64!!!", "k1"},
		"entry wrong key size":    {"k1:" + base64.RawStdEncoding.EncodeToString([]byte("short")), "k1"},
		"empty entry":             {"k1:" + validA + ",", "k1"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ENCRYPTION_KEY", "")
			t.Setenv("ENCRYPTION_KEYS", tc.keys)
			t.Setenv("ENCRYPTION_ACTIVE_KEY_ID", tc.active)
			t.Setenv("ENCRYPTION_KEY_OLD", "")

			defer func() {
				if recover() == nil {
					t.Errorf("expected InitializeEncryption to panic for case %q", name)
				}
			}()

			InitializeEncryption()
		})
	}
}
