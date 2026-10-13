package util

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func setTestKey32() {
	// 32-byte key for AES-256
	bin, _ := base64.RawStdEncoding.DecodeString("mD0I8u2luw52GIQpEteYWQu2UxsWP4kacSBhjgAh5C9")
	key = string(bin)
	keyPresent = true
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setTestKey32()

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

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt(%q) failed: %v", pw, err)
		}

		if decrypted != pw {
			t.Errorf("round-trip failed for %q: got %q", pw, decrypted)
		}
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	setTestKey32()

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
	setTestKey32()

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

func TestDecryptLegacyCBC(t *testing.T) {
	setTestKey32()

	// This ciphertext was produced by the old CBC zero-IV encoder with the
	// same 32-byte test key for plaintext "sources-api-tests".
	legacy := "f80zhczL8GTPqCdlU8WX7m+8BgCZgwXERNGYUF7J+lU"

	out, err := Decrypt(legacy)
	if err != nil {
		t.Fatalf("legacy CBC Decrypt failed: %v", err)
	}

	if out != "sources-api-tests" {
		t.Errorf("legacy CBC decryption: got %q, want %q", out, "sources-api-tests")
	}
}

// TestDecryptCBCCollisionWithSchemeByte verifies that a legacy CBC-encrypted
// password whose raw ciphertext happens to start with byte 0x01 (the GCM
// scheme marker) is still decryptable. Decrypt must try GCM first and, on
// authentication failure, fall back to the CBC decoder.
func TestDecryptCBCCollisionWithSchemeByte(t *testing.T) {
	setTestKey32()

	// "000428-cbc-test" encrypted with the old CBC zero-IV scheme and the
	// 32-byte test key produces a ciphertext whose first decoded byte is
	// 0x01 — the same value as schemeGCM.  Without the GCM-fallback logic
	// this would be misrouted to decodeGCM and fail.
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

func TestDecryptBadFormat(t *testing.T) {
	setTestKey32()

	_, err := Decrypt("a bad thing")
	if err == nil {
		t.Error("expected an error for bad base64 input")
	}

	_, err = Decrypt("this is not a real string")
	if err == nil {
		t.Error("expected an error for invalid ciphertext")
	}
}

func TestDecryptTamperedGCM(t *testing.T) {
	setTestKey32()

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

	_, err = Decrypt(tampered)
	if err == nil {
		t.Error("expected GCM decryption to fail on tampered ciphertext")
	}
}

func TestNoKey(t *testing.T) {
	key = ""
	keyPresent = false

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

func TestSetDefaultEncryptionKey(t *testing.T) {
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
	key = "aaaaaaaaaaaaaaaa"
	keyPresent = true

	pw := "test-with-short-key"

	enc, err := Encrypt(pw)
	if err != nil {
		t.Fatalf("Encrypt with 16-byte key failed: %v", err)
	}

	dec, err := Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt with 16-byte key failed: %v", err)
	}

	if dec != pw {
		t.Errorf("round-trip with 16-byte key: got %q, want %q", dec, pw)
	}
}
