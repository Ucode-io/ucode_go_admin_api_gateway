package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"ucode/ucode_go_api_gateway/pkg/secureenv"

	"github.com/joho/godotenv"
)

const encryptedMetaAdsEnvPath = "/meta-ads.env.enc"

func loadRuntimeEnvironment() {
	loaded := false
	for _, path := range []string{"/app/.env", ".env"} {
		if err := godotenv.Load(path); err == nil {
			loaded = true
		}
	}

	if !loaded {
		log.Println("No .env file found")
	}

	if err := loadEncryptedMetaAdsEnvironment(encryptedMetaAdsEnvPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Encrypted Meta Ads environment was not loaded: %v", err)
	}
}

func loadEncryptedMetaAdsEnvironment(path string) error {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(encoded)) == "" {
		return nil
	}

	secretKey := strings.TrimSpace(os.Getenv("SECRET_KEY"))
	if secretKey == "" {
		return errors.New("SECRET_KEY is required to decrypt Meta Ads configuration")
	}

	plaintext, err := secureenv.Decrypt(strings.TrimSpace(string(encoded)), secretKey)
	if err != nil {
		return errors.New("encrypted Meta Ads configuration is invalid")
	}

	values, err := godotenv.Unmarshal(string(plaintext))
	if err != nil {
		return errors.New("decrypted Meta Ads configuration is invalid")
	}

	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return nil
}
