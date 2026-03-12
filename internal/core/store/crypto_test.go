package store_test

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/justestif/specto/internal/core/store"
)

// validKey returns a random 32-byte hex-encoded key for tests.
func validKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return hex.EncodeToString(key)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := validKey(t)
	plaintext := []byte(`{"access_token":"tok-123","refresh_token":"ref-456"}`)

	ciphertext, err := store.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should differ from plaintext")
	}

	got, err := store.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if string(got) != string(plaintext) {
		t.Errorf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesUniqueCiphertext(t *testing.T) {
	key := validKey(t)
	plaintext := []byte("same input")

	ct1, err := store.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("first Encrypt() error: %v", err)
	}

	ct2, err := store.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("second Encrypt() error: %v", err)
	}

	// Different nonces should produce different ciphertexts.
	if string(ct1) == string(ct2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (unique nonces)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := validKey(t)
	key2 := validKey(t)

	ciphertext, err := store.Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	_, err = store.Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("Decrypt() with wrong key should fail")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := validKey(t)

	ciphertext, err := store.Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	// Flip a byte in the ciphertext (past the nonce).
	if len(ciphertext) > 13 {
		ciphertext[13] ^= 0xff
	}

	_, err = store.Decrypt(ciphertext, key)
	if err == nil {
		t.Fatal("Decrypt() with tampered ciphertext should fail")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := validKey(t)

	_, err := store.Decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("Decrypt() with too-short ciphertext should fail")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := validKey(t)

	ciphertext, err := store.Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	got, err := store.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Decrypt() = %q, want empty", got)
	}
}

func TestInvalidHexKey(t *testing.T) {
	_, err := store.Encrypt([]byte("test"), "not-hex")
	if err == nil {
		t.Fatal("Encrypt() with invalid hex key should fail")
	}
}

func TestKeyWrongLength(t *testing.T) {
	shortKey := hex.EncodeToString(make([]byte, 16)) // 16 bytes, need 32

	_, err := store.Encrypt([]byte("test"), shortKey)
	if err == nil {
		t.Fatal("Encrypt() with 16-byte key should fail (need 32)")
	}
}

func TestLargePlaintext(t *testing.T) {
	key := validKey(t)
	// 1 MB of data
	plaintext := make([]byte, 1<<20)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := store.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	got, err := store.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if len(got) != len(plaintext) {
		t.Fatalf("Decrypt() length = %d, want %d", len(got), len(plaintext))
	}
	for i := range plaintext {
		if got[i] != plaintext[i] {
			t.Fatalf("Decrypt() mismatch at byte %d", i)
		}
	}
}
