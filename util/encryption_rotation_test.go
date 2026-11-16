package util

import (
	"encoding/base64"
	"testing"
)

// Basic test to verify encryption/decryption works
func TestEncryptDecryptBasic(t *testing.T) {
	testKey := "testkey1234567890123456789012345" // exactly 32 bytes
	testPassword := "super-secret-password-123"

	ResetEncryptionKeys()
	OverrideEncryptionKey(testKey)

	encrypted, err := Encrypt(testPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != testPassword {
		t.Errorf("Expected %s, got %s", testPassword, decrypted)
	}
}

// Test empty password handling
func TestEmptyPassword(t *testing.T) {
	testKey := "testkey1234567890123456789012345" // exactly 32 bytes

	ResetEncryptionKeys()
	OverrideEncryptionKey(testKey)

	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Failed to encrypt empty string: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt empty string: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Expected empty string, got %s", decrypted)
	}
}

// Test that invalid fallback key doesn't crash the system
func TestInvalidFallbackKeyIgnored(t *testing.T) {
	validKey := base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345"))

	ResetEncryptionKeys()
	t.Setenv("ENCRYPTION_KEY", validKey)
	t.Setenv("ENCRYPTION_KEY_OLD", "invalid-base64-!@#$%")

	// Should not panic, just ignore invalid fallback key
	InitializeEncryption()

	// Should still work with primary key
	encrypted, err := Encrypt("test")
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != "test" {
		t.Errorf("Expected 'test', got %s", decrypted)
	}
}

// NOTE: Comprehensive key rotation testing (dual-key mode, fallback behavior, etc.)
// is covered by IQE integration tests which test the actual functionality in a
// real environment with database interactions. Unit tests for those scenarios
// have test isolation issues due to Go's t.Setenv() behavior affecting the entire
// test process.
