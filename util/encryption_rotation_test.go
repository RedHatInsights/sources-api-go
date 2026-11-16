package util

import (
	"encoding/base64"
	"testing"
)

func TestDualKeyRotationSupport(t *testing.T) {
	// Generate two different test keys (exactly 32 bytes for AES-256)
	oldKey := "oldkey12345678901234567890123456" // exactly 32 bytes
	newKey := "newkey12345678901234567890123456" // exactly 32 bytes

	oldKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(oldKey))
	newKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(newKey))

	testPassword := "super-secret-password-123"

	// Pre-encrypt test data with old key BEFORE any subtests run
	// This avoids state pollution between tests
	ResetEncryptionKeys()
	OverrideEncryptionKey(oldKey)
	dataEncryptedWithOldKey, _ := Encrypt(testPassword)

	// Pre-encrypt data for "new key" test
	ResetEncryptionKeys()
	OverrideEncryptionKey(newKey)
	dataEncryptedWithNewKey, _ := Encrypt(testPassword)

	t.Run("Encrypt and decrypt with new key only", func(t *testing.T) {
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		InitializeEncryption()

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
	})

	t.Run("Decrypt old data with fallback key", func(t *testing.T) {
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
		InitializeEncryption()

		decrypted, err := Decrypt(dataEncryptedWithOldKey)
		if err != nil {
			t.Fatalf("Failed to decrypt old data with fallback key: %v", err)
		}

		if decrypted != testPassword {
			t.Errorf("Expected %s, got %s", testPassword, decrypted)
		}
	})

	t.Run("New encryptions always use new key", func(t *testing.T) {
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
		InitializeEncryption()

		// Encrypt with dual-key mode active
		encrypted, err := Encrypt(testPassword)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Verify it can be decrypted with ONLY the new key (no fallback)
		// by using pre-encrypted data that we know was encrypted with new key
		ResetEncryptionKeys()
		OverrideEncryptionKey(newKey)

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt with new key only: %v", err)
		}

		if decrypted != testPassword {
			t.Errorf("Expected %s, got %s", testPassword, decrypted)
		}
	})

	t.Run("Cannot decrypt old data without fallback key", func(t *testing.T) {
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		InitializeEncryption()

		_, err := Decrypt(dataEncryptedWithOldKey)
		if err == nil {
			t.Error("Expected decryption to fail without fallback key, but it succeeded")
		}
	})

	t.Run("Override key functionality still works", func(t *testing.T) {
		ResetEncryptionKeys()
		customKey := "12345678901234567890123456789012" // exactly 32 bytes
		OverrideEncryptionKey(customKey)

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
	})
}

func TestDualKeyRotationEdgeCases(t *testing.T) {
	t.Run("Empty password", func(t *testing.T) {
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345")))
		InitializeEncryption()

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
	})

	t.Run("Invalid fallback key is ignored gracefully", func(t *testing.T) {
		ResetEncryptionKeys()
		validKey := base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345"))

		t.Setenv("ENCRYPTION_KEY", validKey)
		t.Setenv("ENCRYPTION_KEY_OLD", "invalid-base64-!@#$%")

		// Should not panic, just ignore the invalid fallback key
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
	})
}
