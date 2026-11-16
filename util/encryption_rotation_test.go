package util

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptNewKeyOnly(t *testing.T) {
	newKey := "newkey12345678901234567890123456" // exactly 32 bytes
	newKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(newKey))
	testPassword := "super-secret-password-123"

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
}

func TestDecryptOldDataWithFallbackKey(t *testing.T) {
	oldKey := "oldkey12345678901234567890123456" // exactly 32 bytes
	newKey := "newkey12345678901234567890123456" // exactly 32 bytes

	oldKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(oldKey))
	newKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(newKey))
	testPassword := "super-secret-password-123"

	// First, encrypt with old key
	ResetEncryptionKeys()
	OverrideEncryptionKey(oldKey)
	encryptedWithOldKey, _ := Encrypt(testPassword)

	// Now setup dual-key mode
	ResetEncryptionKeys()
	t.Setenv("ENCRYPTION_KEY", newKeyB64)
	t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
	InitializeEncryption()

	// Should decrypt old data using fallback key
	decrypted, err := Decrypt(encryptedWithOldKey)
	if err != nil {
		t.Fatalf("Failed to decrypt old data with fallback key: %v", err)
	}

	if decrypted != testPassword {
		t.Errorf("Expected %s, got %s", testPassword, decrypted)
	}
}

func TestNewEncryptionsAlwaysUseNewKey(t *testing.T) {
	newKey := "newkey12345678901234567890123456" // exactly 32 bytes
	oldKey := "oldkey12345678901234567890123456" // exactly 32 bytes

	oldKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(oldKey))
	newKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(newKey))
	testPassword := "super-secret-password-123"

	// Setup dual-key mode
	ResetEncryptionKeys()
	t.Setenv("ENCRYPTION_KEY", newKeyB64)
	t.Setenv("ENCRYPTION_KEY_OLD", oldKeyB64)
	InitializeEncryption()

	// Encrypt with dual-key mode
	encrypted, err := Encrypt(testPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Verify it decrypts with ONLY new key (no fallback)
	ResetEncryptionKeys()
	OverrideEncryptionKey(newKey)

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt with new key only: %v", err)
	}

	if decrypted != testPassword {
		t.Errorf("Expected %s, got %s", testPassword, decrypted)
	}
}

func TestCannotDecryptOldDataWithoutFallbackKey(t *testing.T) {
	oldKey := "oldkey12345678901234567890123456" // exactly 32 bytes
	newKey := "newkey12345678901234567890123456" // exactly 32 bytes

	newKeyB64 := base64.RawStdEncoding.EncodeToString([]byte(newKey))
	testPassword := "super-secret-password-123"

	// Encrypt with old key
	ResetEncryptionKeys()
	OverrideEncryptionKey(oldKey)
	encryptedWithOldKey, _ := Encrypt(testPassword)

	// Try to decrypt with only new key (no fallback)
	ResetEncryptionKeys()
	t.Setenv("ENCRYPTION_KEY", newKeyB64)
	InitializeEncryption()

	_, err := Decrypt(encryptedWithOldKey)
	if err == nil {
		t.Error("Expected decryption to fail without fallback key, but it succeeded")
	}
}

func TestOverrideKeyFunctionality(t *testing.T) {
	testPassword := "super-secret-password-123"

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
}

func TestEmptyPassword(t *testing.T) {
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
}

func TestInvalidFallbackKeyIgnored(t *testing.T) {
	ResetEncryptionKeys()
	validKey := base64.RawStdEncoding.EncodeToString([]byte("testkey1234567890123456789012345"))

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
