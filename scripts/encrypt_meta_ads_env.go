package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"ucode/ucode_go_api_gateway/pkg/secureenv"
)

func main() {
	if len(os.Args) != 3 {
		fatal(errors.New("usage: encrypt_meta_ads_env <base-env-path> <output-path>"))
	}

	secretKey, err := readEnvValue(os.Args[1], "SECRET_KEY")
	if err != nil {
		fatal(err)
	}

	accessToken := strings.TrimSpace(os.Getenv("META_ACCESS_TOKEN"))
	if accessToken == "" {
		fatal(errors.New("META_ACCESS_TOKEN is required"))
	}
	if strings.ContainsAny(accessToken, "\r\n") {
		fatal(errors.New("META_ACCESS_TOKEN contains an invalid line break"))
	}

	plaintext := strings.Join([]string{
		"META_ACCESS_TOKEN=" + accessToken,
		"META_AD_ACCOUNT_ID=843587364827107",
		"META_GRAPH_BASE_URL=https://graph.facebook.com",
		"META_GRAPH_VERSION=v26.0",
		"META_ADS_CACHE_TTL_SEC=1800",
		"META_ADS_REQUEST_TIMEOUT_SEC=30",
		"META_ADS_MAX_RANGE_DAYS=90",
		"META_LEAD_ACTION_TYPES=lead",
		"META_ATTRIBUTION_WINDOWS=",
		"",
	}, "\n")

	encoded, err := secureenv.Encrypt([]byte(plaintext), secretKey)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(os.Args[2], []byte(encoded+"\n"), 0o600); err != nil {
		fatal(err)
	}

	fmt.Println("Encrypted Meta Ads configuration created.")
}

func readEnvValue(path, wantedKey string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok && strings.TrimSpace(key) == wantedKey {
			value = strings.TrimSpace(value)
			if value == "" {
				return "", fmt.Errorf("%s is empty", wantedKey)
			}
			return value, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s was not found in %s", wantedKey, path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
