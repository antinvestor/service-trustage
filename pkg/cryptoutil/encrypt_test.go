// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
