//nolint:testpackage // package-local tests exercise unexported crypto helpers intentionally.
package cryptoutil

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncryptDecryptAndTokenHelpers(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte("k"), 32)
	plaintext := []byte("secret-value")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Decrypt() = %q", decrypted)
	}

	if _, decryptErr := Decrypt(key, []byte("short")); decryptErr == nil {
		t.Fatal("expected short ciphertext error")
	}

	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if len(token) != tokenLength*2 {
		t.Fatalf("token length = %d", len(token))
	}

	hash1 := HashToken("abc")
	hash2 := HashToken("abc")
	if hash1 != hash2 || len(hash1) != 64 || strings.Trim(hash1, "0123456789abcdef") != "" {
		t.Fatalf("HashToken() = %q", hash1)
	}
}
