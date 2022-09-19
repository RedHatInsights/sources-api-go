package util

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"testing"
)

func TestEncrypt(t *testing.T) {
	bin, _ := base64.RawStdEncoding.DecodeString("mD0I8u2luw52GIQpEteYWQu2UxsWP4kacSBhjgAh5C9")
	key = string(bin)
	keyPresent = true

	out, err := Encrypt("sources-api-tests")
	if err != nil {
		t.Error(err)
	}

	if out != "f80zhczL8GTPqCdlU8WX7m+8BgCZgwXERNGYUF7J+lU" {
		t.Errorf("encryption failed, got %v expected %v", out, "f80zhczL8GTPqCdlU8WX7m+8BgCZgwXERNGYUF7J+lU")
	}
}

func TestDecrypt(t *testing.T) {
	bin, _ := base64.RawStdEncoding.DecodeString("mD0I8u2luw52GIQpEteYWQu2UxsWP4kacSBhjgAh5C9")
	key = string(bin)
	keyPresent = true

	out, err := Decrypt("f80zhczL8GTPqCdlU8WX7m+8BgCZgwXERNGYUF7J+lU")
	if err != nil {
		t.Error(err)
	}

	if out != "sources-api-tests" {
		t.Errorf("decryption failed, got %v expected %v", out, "sources-api-tests")
	}
}

func TestBadFormat(t *testing.T) {
	bin, _ := base64.RawStdEncoding.DecodeString("mD0I8u2luw52GIQpEteYWQu2UxsWP4kacSBhjgAh5C9")
	key = string(bin)

	_, err := Decrypt("a bad thing")
	if err == nil {
		t.Errorf("'a bad thing': expected an err but none was returned")
	}

	_, err = Decrypt("this is not a real string")
	if err == nil {
		t.Errorf("expected an err but none was returned")
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

func TestNoKey(t *testing.T) {
	key = ""
	keyPresent = false

	_, err := Decrypt("another thing")
	if err == nil {
		t.Errorf("expected an err but none was returned")
	}

	if err.Error() != "no encryption key present" {
		t.Errorf("bad error message: %v", err.Error())
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
func TestInitEncryptionNoKey(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("execution should have panicked but did not")
		}
		f, fErr := os.Create("encryption_key.dev")
		if fErr != nil {
			t.Errorf("file encryption_key.dev was not able to be created")
		}
		_, err2 := f.WriteString("YWFhYWFhYWFhYWFhYWFhYQ")
		if err2 != nil {
			t.Errorf("Was unable to write key to file.")
		}
		closeErr := f.Close()
		if closeErr != nil {
			t.Errorf("Unable to close written file")
		}
		symErr := os.Symlink("encryption_key.dev", "encryption_key")
		if symErr != nil {
			t.Errorf("Error within symlink")
		}

		body, _ := ioutil.ReadFile("encryption_key.dev")
		if string(body) != "YWFhYWFhYWFhYWFhYWFhYQ" {
			t.Errorf("Wrong default key")
		}
	}()
	e := os.Remove("encryption_key.dev")
	_ = os.Setenv("ENCRYPTION_KEY", "")
	if e != nil {
		panic(e)
	}
	if _, sErr := os.Lstat("encryption_key"); sErr == nil {
		if sErr := os.Remove("encryption_key"); sErr != nil {
			t.Errorf("Failed to unlink: %v", sErr)
		}
	} else if os.IsNotExist(sErr) {
		t.Errorf("Failed to check symlink: %v", sErr)
	}
	InitializeEncryption()
}
