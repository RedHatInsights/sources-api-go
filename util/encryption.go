package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// "the key" to encrypt/decrypt passwords with
	key        = os.Getenv("ENCRYPTION_KEY")
	keyPresent = false

	// regex to pull the actual secret out of the miq-password style column
	xtractRe = regexp.MustCompile(`v2:{(.+)}`)
)

// init for this file basically just decodes the ENCRYPTION_KEY env var to the
// plain text equivalent
func init() {
	if key == "" {
		return
	}

	// the key is base64 encoded in the ENV
	decoded, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil {
		panic(err)
	}
	key = string(decoded)
	keyPresent = true
}

// Encrypts str into a miq-compatible password format using the encryption key
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
	return "v2:{" + base64.RawStdEncoding.EncodeToString([]byte(encoded)) + "}", nil
}

// Decrypts a miq-compatible password into a string
func Decrypt(str string) (string, error) {
	if !keyPresent {
		return "", fmt.Errorf("no encryption key present")
	}

	if !xtractRe.Match([]byte(str)) {
		return "", fmt.Errorf("string does not match format")
	}

	// extract the password from between the brackets
	matches := xtractRe.FindAllStringSubmatch(str, 1)

	// the password is base64 encoded
	rawPass, err := base64.RawStdEncoding.DecodeString(matches[0][1])
	if err != nil {
		return "", err
	}

	return decode(string(rawPass))
}

func decode(pw string) (string, error) {
	// create the block from the key
	block, err := aes.NewCipher([]byte(key))
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
