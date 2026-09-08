package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/RedHatInsights/sources-api-go/config"
)

const (
	// schemeGCM is the version byte prepended to AES-GCM ciphertexts.
	// Format: 0x01 || nonce(12) || ciphertext || tag(16)
	schemeGCM byte = 0x01
)

var (
	// "the key" to encrypt/decrypt passwords with
	key        string
	keyPresent = false
)

// InitializeEncryption allows reinitializing the encryption key by reading from the "ENCRYPTION_KEY" environment
// variable again. Useful for testing purposes outside the "util" package.
func InitializeEncryption() {
	key = os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		var err error

		key, err = setDefaultEncryptionKey()
		if err != nil {
			panic(err)
		}
	}
	// the key is base64 encoded in the ENV
	decoded, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil {
		panic(err)
	}

	key = string(decoded)
	keyPresent = true
}

func setDefaultEncryptionKey() (string, error) {
	if config.Get().Env != "stage" && config.Get().Env != "prod" {
		// fetch the file name so we know where to get the source encryption_key.dev file from
		_, filename, _, ok := runtime.Caller(1)
		if !ok {
			return "", fmt.Errorf("not possible to recover the information")
		}

		filepath := path.Join(path.Dir(filename), "encryption_key.dev")

		_, err := os.Stat(filepath)
		if os.IsNotExist(err) {
			panic("encryption_key.dev file does not exist. Please import encryption_key.dev file and create symlink encryption_key.dev to encryption_key using [ln -s FILE LINK].")
		} else {
			body, bodyErr := os.ReadFile(filepath)
			if bodyErr != nil {
				return "", fmt.Errorf("unable to read file: %v", err)
			}

			key = string(body)

			keyErr := os.Setenv("ENCRYPTION_KEY", string(body))
			if keyErr != nil {
				return "", fmt.Errorf("error in setting variable in environment: %v", keyErr)
			}

			return key, nil
		}
	}

	panic("Unable to set up default encryption key")
}

// Encrypt encrypts str using AES-256-GCM with a random nonce. The output is a
// base64-encoded blob prefixed with a scheme byte so that Decrypt can
// distinguish it from legacy CBC ciphertexts during migration.
func Encrypt(str string) (string, error) {
	if !keyPresent {
		return "", fmt.Errorf("no encryption key present")
	}

	raw, err := encodeGCM(str)
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(raw), nil
}

// Decrypt decrypts a password. It transparently handles both the new AES-GCM
// format (scheme byte 0x01) and the legacy AES-CBC zero-IV format so that
// existing rows are readable without a bulk migration.
func Decrypt(str string) (string, error) {
	if !keyPresent {
		return "", fmt.Errorf("no encryption key present")
	}

	rawPass, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	if len(rawPass) > 0 && rawPass[0] == schemeGCM {
		return decodeGCM(rawPass)
	}

	// Legacy CBC path — existing rows encrypted with the old zero-IV scheme.
	return decodeLegacyCBC(rawPass)
}

// encodeGCM encrypts plaintext with AES-GCM using a random 12-byte nonce.
// Returns: schemeGCM(1) || nonce(12) || ciphertext+tag.
func encodeGCM(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// 1 byte scheme + nonce + ciphertext (includes GCM tag)
	result := make([]byte, 1+len(nonce)+len(ciphertext))
	result[0] = schemeGCM
	copy(result[1:], nonce)
	copy(result[1+len(nonce):], ciphertext)

	return result, nil
}

// decodeGCM decrypts an AES-GCM ciphertext produced by encodeGCM.
// Expects: schemeGCM(1) || nonce(12) || ciphertext+tag.
func decodeGCM(data []byte) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()

	minLen := 1 + nonceSize + gcm.Overhead()
	if len(data) < minLen {
		return "", fmt.Errorf("ciphertext too short for AES-GCM: need at least %d bytes, got %d", minLen, len(data))
	}

	nonce := data[1 : 1+nonceSize]
	ciphertext := data[1+nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// decodeLegacyCBC decrypts data encrypted with the old AES-CBC zero-IV scheme.
// Kept for backward compatibility so existing database rows can still be read.
// These rows will be re-encrypted with AES-GCM on the next write.
func decodeLegacyCBC(data []byte) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	if len(data)%block.BlockSize() != 0 {
		return "", fmt.Errorf("ciphertext length %d is not a multiple of block size %d", len(data), block.BlockSize())
	}

	cbc := cipher.NewCBCDecrypter(block, make([]byte, block.BlockSize()))

	output := make([]byte, len(data))
	cbc.CryptBlocks(output, data)

	return strings.Trim(string(output), "\x00"), nil
}

func OverrideEncryptionKey(k string) {
	key = k
	keyPresent = true
}
