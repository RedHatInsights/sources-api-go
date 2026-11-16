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

	// Test password
	testPassword := "super-secret-password-123"

	t.Run("Encrypt and decrypt with new key only", func(t *testing.T) {
		// Reset encryption state
		ResetEncryptionKeys()

		// Setup: only new key present (t.Setenv auto-cleans up after test)
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		InitializeEncryption()

		// Encrypt with new key
		encrypted, err := Encrypt(testPassword)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt with new key
		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		if decrypted != testPassword {
			t.Errorf("Expected %s, got %s", testPassword, decrypted)
		}
	})

	t.Run("Decrypt old data with fallback key", func(t *testing.T) {
		// Reset encryption state
		ResetEncryptionKeys()

		// First, encrypt with old key
		t.Setenv("ENCRYPTION_KEY", oldKeyB64)
		InitializeEncryption()

		encryptedWithOldKey, err := Encrypt(testPassword)
		if err != nil {
			t.Fatalf("Failed to encrypt with old key: %v", err)
		}

		// Now setup dual-key mode (new primary, old fallback)
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
		InitializeEncryption()

		// Should be able to decrypt old data
		decrypted, err := Decrypt(encryptedWithOldKey)
		if err != nil {
			t.Fatalf("Failed to decrypt old data with fallback key: %v", err)
		}

		if decrypted != testPassword {
			t.Errorf("Expected %s, got %s", testPassword, decrypted)
		}
	})

	t.Run("New encryptions always use new key", func(t *testing.T) {
		// Reset encryption state
		ResetEncryptionKeys()

		// Setup dual-key mode
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
		InitializeEncryption()

		// Encrypt something
		encrypted, err := Encrypt(testPassword)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Setup with only new key (no fallback)
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		InitializeEncryption()

		// Should still be able to decrypt (proves it was encrypted with new key)
		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt with new key only: %v", err)
		}

		if decrypted != testPassword {
			t.Errorf("Expected %s, got %s", testPassword, decrypted)
		}
	})

	t.Run("Cannot decrypt old data without fallback key", func(t *testing.T) {
		// Reset encryption state
		ResetEncryptionKeys()

		// Encrypt with old key
		t.Setenv("ENCRYPTION_KEY", oldKeyB64)
		InitializeEncryption()

		encryptedWithOldKey, err := Encrypt(testPassword)
		if err != nil {
			t.Fatalf("Failed to encrypt with old key: %v", err)
		}

		// Setup with only new key (no fallback)
		ResetEncryptionKeys()
		t.Setenv("ENCRYPTION_KEY", newKeyB64)
		InitializeEncryption()

		// Should fail to decrypt old data
		_, err = Decrypt(encryptedWithOldKey)
		if err == nil {
			t.Error("Expected decryption to fail without fallback key, but it succeeded")
		}
	})

	t.Run("Override key functionality still works", func(t *testing.T) {
		// Reset encryption state
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
		// Reset encryption state
		ResetEncryptionKeys()

		t.Setenv("ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345"))) // exactly 32 bytes
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
		// Reset encryption state
		ResetEncryptionKeys()

		validKey := base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345")) // exactly 32 bytes

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
