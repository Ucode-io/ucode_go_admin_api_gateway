package secureenv

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	version = "v1"
	context = "ucode/meta-ads-env/v1"
)

func Encrypt(plaintext []byte, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("encryption secret is required")
	}

	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := gcm.Seal(nil, nonce, plaintext, []byte(context))
	payload := append(nonce, sealed...)
	return version + ":" + base64.RawStdEncoding.EncodeToString(payload), nil
}

func Decrypt(encoded, secret string) ([]byte, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("decryption secret is required")
	}

	encodedVersion, payloadText, ok := strings.Cut(strings.TrimSpace(encoded), ":")
	if !ok || encodedVersion != version {
		return nil, errors.New("unsupported encrypted payload version")
	}

	payload, err := base64.RawStdEncoding.DecodeString(payloadText)
	if err != nil {
		return nil, errors.New("invalid encrypted payload encoding")
	}

	gcm, err := newGCM(secret)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, errors.New("encrypted payload is too short")
	}

	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(context))
	if err != nil {
		return nil, errors.New("encrypted payload authentication failed")
	}
	return plaintext, nil
}

func newGCM(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(context + "\x00" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
