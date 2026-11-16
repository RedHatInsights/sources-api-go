package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/RedHatInsights/sources-api-go/config"
)

var (
	// "the key" to encrypt/decrypt passwords with (primary/new key)
	key        string
	keyPresent = false

	// fallback key for reading old encrypted data during rotation
	fallbackKey        string
	fallbackKeyPresent = false
)

// InitializeEncryption allows reinitializing the encryption key by reading from the "ENCRYPTION_KEY" environment
// variable again. Useful for testing purposes outside the "util" package.
func InitializeEncryption() {
	// Reset fallback key state on each initialization
	fallbackKey = ""
	fallbackKeyPresent = false

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

	// Load fallback key for rotation support (optional)
	fallbackKeyStr := os.Getenv("ENCRYPTION_KEY_OLD")
	if fallbackKeyStr != "" {
		decodedFallback, err := base64.RawStdEncoding.DecodeString(fallbackKeyStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to decode fallback encryption key: %v\n", err)
		} else {
			fallbackKey = string(decodedFallback)
			fallbackKeyPresent = true
			fmt.Fprintf(os.Stderr, "Info: Fallback encryption key loaded for dual-key rotation support\n")
		}
	}
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

// Encrypts str into a password_hash using the encryption key
// in the environment
func Encrypt(str string) (string, error) {
	if !keyPresent {
		return "", fmt.Errorf("no encryption key present")
	}

	encoded, err := encode(str)
	if err != nil {
		return "", err
	}

	// base64 encode the encrypted secret for text-storage
	return base64.RawStdEncoding.EncodeToString([]byte(encoded)), nil
}

// Decrypts a password into a string
func Decrypt(str string) (string, error) {
	if !keyPresent {
		return "", fmt.Errorf("no encryption key present")
	}

	// the password is base64 encoded
	rawPass, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	// Try decrypting with the new/primary key first
	decrypted, err := decodeWithKey(string(rawPass), key)
	if err == nil {
		return decrypted, nil
	}

	// If fallback key is present, try decrypting with the old key
	if fallbackKeyPresent {
		decrypted, err = decodeWithKey(string(rawPass), fallbackKey)
		if err == nil {
			// Log for monitoring - this indicates data still encrypted with old key
			fmt.Fprintf(os.Stderr, "Info: Decrypted data using fallback key\n")
			return decrypted, nil
		}
	}

	return "", fmt.Errorf("failed to decrypt with available keys")
}

func decode(pw string) (string, error) {
	return decodeWithKey(pw, key)
}

func decodeWithKey(pw string, encKey string) (string, error) {
	// create the block from the key
	block, err := aes.NewCipher([]byte(encKey))
	if err != nil {
		return "", err
	}

	// no iv used, but need an iv that is the same length as the block size.
	cbc256 := cipher.NewCBCDecrypter(block, make([]byte, block.BlockSize()))

	// output size must be the same as input. using the same bytes for
	// input/output is fine but can lead to trailing spaces which is why we trim
	// it before returning
	output := make([]byte, len(pw))
	cbc256.CryptBlocks(output, []byte(pw))

	// strip out the trailing zeros that come from the extra length due to the
	// block size
	return strings.Trim(string(output), "\x00"), nil
}

func encode(pw string) (string, error) {
	// create the block from the key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// no iv used, but need an iv that is the same length as the block size.
	cbc256 := cipher.NewCBCEncrypter(block, make([]byte, block.BlockSize()))

	// create a plain text representation at the right length, and an output
	// slice the same length
	plaintext := padString(pw, block.BlockSize())
	output := make([]byte, len(plaintext))

	// encrypt the string into the output byte array
	cbc256.CryptBlocks(output, []byte(plaintext))

	// base64 encode it and return!
	return string(output), nil
}

// helper function to create a string that is the "proper" length to be
// encrypted. the right block size is basically the length of the string +
// however many bytes it takes to get up to a multiple of the blocksize
//
// in mathy terms: length = len(password) + (len(password) % blocks)
//
// e.g. a string length 4 with a blocksize of 8 would return the string with
// four spaces at the end.
func padString(text string, blockSize int) string {
	if blockSize < 0 {
		panic("negative blocksize")
	}

	if len(text) == blockSize {
		return text
	}

	padLength := blockSize - len(text)%blockSize
	padding := bytes.Repeat([]byte{byte(0)}, padLength)

	return text + string(padding)
}

func OverrideEncryptionKey(k string) {
	key = k
	keyPresent = true
}

// ResetEncryptionKeys clears the encryption keys (for testing only)
func ResetEncryptionKeys() {
	key = ""
	keyPresent = false
	fallbackKey = ""
	fallbackKeyPresent = false
}
