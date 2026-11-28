package util

import (
	"bytes"
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

	// encryptionCanary is the probe string round-tripped at initialization
	// time to verify the loaded keys are usable before serving traffic.
	encryptionCanary = "sources-api-encryption-canary"

	// defaultEncryptionKeyID is the id synthesized for the single key loaded
	// via the ENCRYPTION_KEY shorthand (or the dev key file). Rows written
	// while running the shorthand are stamped with this id, so when a full
	// ENCRYPTION_KEYS keyring is adopted later this id must map to that same
	// key (or those rows must be rotated first).
	defaultEncryptionKeyID = "v1"
)

var (
	// encryptionKeys is the keyring: key id -> raw key bytes. New ciphertext
	// is always encrypted with the active key; the remaining entries exist
	// only to decrypt rows stamped with older key ids. Loaded from
	// ENCRYPTION_KEYS ("id:base64key,..."), or synthesized from the
	// ENCRYPTION_KEY shorthand.
	encryptionKeys = map[string]string{}

	// activeKeyID is the keyring entry new ciphertext is encrypted with,
	// selected by ENCRYPTION_ACTIVE_KEY_ID. It doubles as the "a keyring is
	// loaded" sentinel: empty means uninitialized.
	activeKeyID = ""

	// legacyKey is the pre-rotation key, used only to read — and, during the
	// migration window, keep current — the old "password_hash" column, which
	// holds AES-CBC ciphertext. Loaded from ENCRYPTION_KEY_OLD (optional).
	legacyKey        string
	legacyKeyPresent = false
)

// InitializeEncryption allows reinitializing the encryption keyring by reading from the environment again.
// Useful for testing purposes outside the "util" package.
func InitializeEncryption() {
	legacyKey = ""
	legacyKeyPresent = false

	keys, active, err := loadEncryptionKeyring()
	if err != nil {
		panic(err)
	}

	encryptionKeys = keys
	activeKeyID = active

	// The pre-rotation key is optional: it is only set while legacy CBC rows
	// are being migrated (or dual-written). A key that is set but invalid is a
	// deployment misconfiguration and must fail fast here rather than
	// silently disabling reads of the legacy column at request time.
	if raw := os.Getenv("ENCRYPTION_KEY_OLD"); raw != "" {
		decodedLegacy, err := base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			panic(fmt.Sprintf("failed to decode ENCRYPTION_KEY_OLD (must be base64): %v", err))
		}

		if len(decodedLegacy) == 0 {
			panic("ENCRYPTION_KEY_OLD decoded to an empty key")
		}

		legacyKey = string(decodedLegacy)
		legacyKeyPresent = true
	}

	encryptionSelfTest()
}

// loadEncryptionKeyring loads the keyring from ENCRYPTION_KEYS
// ("id:base64key,id:base64key,..."), with ENCRYPTION_ACTIVE_KEY_ID selecting
// the write key. When ENCRYPTION_KEYS is unset it falls back to the
// ENCRYPTION_KEY shorthand (or the dev key file), synthesizing a single-entry
// keyring under the "v1" id. Every configuration error is returned as an
// error so the caller can fail fast at startup.
func loadEncryptionKeyring() (map[string]string, string, error) {
	keys := map[string]string{}

	if raw := os.Getenv("ENCRYPTION_KEYS"); raw != "" {
		for _, entry := range strings.Split(raw, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				return nil, "", fmt.Errorf("ENCRYPTION_KEYS contains an empty entry")
			}

			id, encoded, found := strings.Cut(entry, ":")
			if !found || id == "" || encoded == "" {
				return nil, "", fmt.Errorf("ENCRYPTION_KEYS entry %q must be of the form id:base64key", entry)
			}

			if _, duplicate := keys[id]; duplicate {
				return nil, "", fmt.Errorf("ENCRYPTION_KEYS contains duplicate key id %q", id)
			}

			key, err := decodeEncryptionKey(encoded)
			if err != nil {
				return nil, "", fmt.Errorf("ENCRYPTION_KEYS entry %q: %w", id, err)
			}

			keys[id] = key
		}
	} else {
		encoded := os.Getenv("ENCRYPTION_KEY")
		if encoded == "" {
			var err error

			encoded, err = setDefaultEncryptionKey()
			if err != nil {
				return nil, "", err
			}
		}

		key, err := decodeEncryptionKey(encoded)
		if err != nil {
			return nil, "", err
		}

		keys[defaultEncryptionKeyID] = key
	}

	if len(keys) == 0 {
		return nil, "", fmt.Errorf("no encryption keys configured")
	}

	active := os.Getenv("ENCRYPTION_ACTIVE_KEY_ID")
	if active == "" {
		if len(keys) != 1 {
			return nil, "", fmt.Errorf("ENCRYPTION_ACTIVE_KEY_ID is required when ENCRYPTION_KEYS holds more than one key")
		}

		for id := range keys {
			active = id
		}
	}

	if _, ok := keys[active]; !ok {
		return nil, "", fmt.Errorf("ENCRYPTION_ACTIVE_KEY_ID %q is not present in the configured encryption keys", active)
	}

	return keys, active, nil
}

// decodeEncryptionKey decodes a base64-encoded (RawStdEncoding, unpadded)
// key and verifies it is a usable AES key size.
func decodeEncryptionKey(encoded string) (string, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("key is not valid unpadded base64: %w", err)
	}

	if len(decoded) == 0 {
		return "", fmt.Errorf("key decoded to zero bytes")
	}

	if !validAESKeySize(len(decoded)) {
		return "", fmt.Errorf("key is %d bytes; AES keys must be 16, 24 or 32 bytes", len(decoded))
	}

	return string(decoded), nil
}

func validAESKeySize(size int) bool {
	return size == 16 || size == 24 || size == 32
}

// encryptionSelfTest round-trips a probe through every keyring entry (and
// the legacy key, when configured) at initialization time, so an unusable
// key stops the process at boot instead of failing per request.
func encryptionSelfTest() {
	for id, key := range encryptionKeys {
		encrypted, err := encodeGCM(encryptionCanary, key)
		if err != nil {
			panic(fmt.Sprintf("encryption self-test failed (encrypt with key %q): %v", id, err))
		}

		decrypted, err := decodeGCM(encrypted, key)
		if err != nil {
			panic(fmt.Sprintf("encryption self-test failed (decrypt with key %q): %v", id, err))
		}

		if decrypted != encryptionCanary {
			panic(fmt.Sprintf("encryption self-test failed: round-trip with key %q returned wrong plaintext", id))
		}
	}

	if legacyKeyPresent {
		legacyEncrypted, err := EncryptLegacyCBC(encryptionCanary)
		if err != nil {
			panic(fmt.Sprintf("encryption self-test failed (legacy CBC encrypt): %v", err))
		}

		legacyDecrypted, err := DecryptLegacyCBC(legacyEncrypted)
		if err != nil {
			panic(fmt.Sprintf("encryption self-test failed (legacy CBC decrypt): %v", err))
		}

		if legacyDecrypted != encryptionCanary {
			panic("encryption self-test failed: legacy CBC round-trip returned wrong plaintext")
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

			keyErr := os.Setenv("ENCRYPTION_KEY", string(body))
			if keyErr != nil {
				return "", fmt.Errorf("error in setting variable in environment: %v", keyErr)
			}

			return string(body), nil
		}
	}

	panic("Unable to set up default encryption key")
}

// Encrypt encrypts str with AES-256-GCM (random nonce) using the active key.
// The output is a base64-encoded blob prefixed with a version byte. New
// ciphertexts only ever land in the "password_hash_gcm" column, stamped with
// the active key id (see ActiveKeyID).
func Encrypt(str string) (string, error) {
	if activeKeyID == "" {
		return "", fmt.Errorf("no encryption key present")
	}

	raw, err := encodeGCM(str, encryptionKeys[activeKeyID])
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(raw), nil
}

// ActiveKeyID returns the id of the key new ciphertext is encrypted with.
// Callers store it next to the ciphertext (Authentication.EncryptionKeyID)
// so later reads know which keyring entry decrypts it.
func ActiveKeyID() string {
	return activeKeyID
}

// DecryptGCM decrypts a base64-encoded AES-GCM ciphertext with the active
// key. Prefer DecryptGCMWithKeyID, which routes by the stored key id; this
// variant exists for tests and diagnostics.
func DecryptGCM(str string) (string, error) {
	if activeKeyID == "" {
		return "", fmt.Errorf("no encryption key present")
	}

	raw, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	return decodeGCM(raw, encryptionKeys[activeKeyID])
}

// DecryptGCMWithKeyID decrypts a base64-encoded AES-GCM ciphertext with the
// keyring entry named by keyID — the id stored next to the ciphertext when
// it was written. An unknown id is a loud error, never a guess: silently
// trying other keys would reintroduce the ambiguity this routing exists to
// remove.
func DecryptGCMWithKeyID(str string, keyID string) (string, error) {
	if activeKeyID == "" {
		return "", fmt.Errorf("no encryption key present")
	}

	key, ok := encryptionKeys[keyID]
	if !ok {
		return "", fmt.Errorf("unknown encryption key id %q — is it missing from ENCRYPTION_KEYS?", keyID)
	}

	raw, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	decryptsTotal.WithLabelValues(keyID).Inc()

	return decodeGCM(raw, key)
}

// LegacyKeyPresent reports whether a pre-rotation key was loaded from
// ENCRYPTION_KEY_OLD. Callers use it to gate dual-writing the legacy column
// during the migration window.
func LegacyKeyPresent() bool {
	return legacyKeyPresent
}

// EncryptLegacyCBC encrypts str with the legacy AES-CBC zero-IV scheme using
// the pre-rotation key. Only used to keep the old "password_hash" column
// current (dual-write) while the migration window is open.
func EncryptLegacyCBC(str string) (string, error) {
	if !legacyKeyPresent {
		return "", fmt.Errorf("no legacy encryption key present (ENCRYPTION_KEY_OLD not set)")
	}

	block, err := aes.NewCipher([]byte(legacyKey))
	if err != nil {
		return "", err
	}

	cbc := cipher.NewCBCEncrypter(block, make([]byte, block.BlockSize()))

	plaintext := padString(str, block.BlockSize())
	output := make([]byte, len(plaintext))
	cbc.CryptBlocks(output, []byte(plaintext))

	return base64.RawStdEncoding.EncodeToString(output), nil
}

// DecryptLegacyCBC decrypts a value from the old "password_hash" column
// (AES-CBC zero-IV, null-byte padded) with the pre-rotation key.
func DecryptLegacyCBC(str string) (string, error) {
	if !legacyKeyPresent {
		return "", fmt.Errorf("no legacy encryption key present (ENCRYPTION_KEY_OLD not set)")
	}

	decryptsTotal.WithLabelValues(legacyKeyIDLabel).Inc()

	raw, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	return decodeLegacyCBCWithKey(legacyKey, raw)
}

// Decrypt decrypts a ciphertext without column or key-id context: AES-GCM
// with the active key if it carries the scheme byte, otherwise the legacy
// CBC scheme. This exists for diagnostics and tests — production code routes
// by the stored key id (see Authentication.GetPassword) and fails loudly
// instead of sniffing.
func Decrypt(str string) (string, error) {
	if activeKeyID == "" {
		return "", fmt.Errorf("no encryption key present")
	}

	raw, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	if len(raw) > 0 && raw[0] == schemeGCM {
		plaintext, gcmErr := decodeGCM(raw, encryptionKeys[activeKeyID])
		if gcmErr == nil {
			return plaintext, nil
		}

		// GCM authentication failed — the legacy CBC ciphertext may happen to
		// start with the same byte value as the GCM scheme marker. Fall
		// through to the CBC decoder so those rows are not broken.
	}

	// Legacy path. With a pre-rotation key loaded, old rows decrypt with it;
	// without one, the only CBC data that can exist was written with the
	// active key (cipher-only migration), so try that.
	if legacyKeyPresent {
		return decodeLegacyCBCWithKey(legacyKey, raw)
	}

	return decodeLegacyCBCWithKey(encryptionKeys[activeKeyID], raw)
}

// encodeGCM encrypts plaintext with AES-GCM using a random 12-byte nonce.
// Returns: schemeGCM(1) || nonce(12) || ciphertext+tag.
func encodeGCM(plaintext string, encKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(encKey))
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
func decodeGCM(data []byte, encKey string) (string, error) {
	block, err := aes.NewCipher([]byte(encKey))
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

// decodeLegacyCBCWithKey decrypts data encrypted with the old AES-CBC
// zero-IV scheme using the given key.
func decodeLegacyCBCWithKey(encKey string, data []byte) (string, error) {
	block, err := aes.NewCipher([]byte(encKey))
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

// OverrideEncryptionKeyring replaces the keyring. Keys are raw bytes (not
// base64). Testing only.
func OverrideEncryptionKeyring(keys map[string]string, activeID string) {
	encryptionKeys = make(map[string]string, len(keys))

	for id, key := range keys {
		encryptionKeys[id] = key
	}

	activeKeyID = activeID
}

// OverrideEncryptionKey overrides the active encryption key under the
// default "v1" id. Testing only.
func OverrideEncryptionKey(k string) {
	OverrideEncryptionKeyring(map[string]string{defaultEncryptionKeyID: k}, defaultEncryptionKeyID)
}

// OverrideLegacyEncryptionKey overrides the pre-rotation key. Testing only.
func OverrideLegacyEncryptionKey(k string) {
	legacyKey = k
	legacyKeyPresent = true
}

// ResetEncryptionKeys clears the keyring and the legacy key. Testing only —
// used to isolate tests from whatever the process environment carried in.
func ResetEncryptionKeys() {
	encryptionKeys = map[string]string{}
	activeKeyID = ""
	legacyKey = ""
	legacyKeyPresent = false
}
