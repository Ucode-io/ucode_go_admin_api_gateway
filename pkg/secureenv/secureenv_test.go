package secureenv

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()

	want := []byte("META_ACCESS_TOKEN=secret\nMETA_AD_ACCOUNT_ID=123\n")
	encoded, err := Encrypt(want, "runtime-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if strings.Contains(encoded, "secret") {
		t.Fatal("encrypted value contains plaintext")
	}

	got, err := Decrypt(encoded, "runtime-secret")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("Decrypt() = %q, want %q", got, want)
	}
}

func TestDecryptRejectsWrongSecretAndTampering(t *testing.T) {
	t.Parallel()

	encoded, err := Encrypt([]byte("META_ACCESS_TOKEN=secret\n"), "runtime-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if _, err := Decrypt(encoded, "wrong-secret"); err == nil {
		t.Fatal("Decrypt() with wrong secret succeeded")
	}

	replacement := "A"
	if encoded[len(encoded)-1:] == replacement {
		replacement = "B"
	}
	tampered := encoded[:len(encoded)-1] + replacement
	if _, err := Decrypt(tampered, "runtime-secret"); err == nil {
		t.Fatal("Decrypt() with tampered payload succeeded")
	}
}
