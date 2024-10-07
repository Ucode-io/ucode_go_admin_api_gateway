package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

func Encrypt(body interface{}, key string) (string, error) {
	respByte, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	keyBytes := []byte(key)
	textBytes := respByte

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(textBytes))
	iv := cipherText[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], textBytes)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func Decrypt(encryptedText, key string) (string, error) {
	keyBytes := []byte(key)

	cipherText, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("cipherText too short")
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return string(cipherText), nil
}
