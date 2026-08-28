package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fatal(errors.New("usage: write_meta_ads_env <output-path>"))
	}

	accessToken := strings.TrimSpace(os.Getenv("META_ACCESS_TOKEN"))
	if accessToken == "" {
		fatal(errors.New("META_ACCESS_TOKEN is required"))
	}
	if strings.ContainsAny(accessToken, "\r\n") {
		fatal(errors.New("META_ACCESS_TOKEN contains an invalid line break"))
	}

	contents := strings.Join([]string{
		"META_ACCESS_TOKEN=" + accessToken,
		"META_AD_ACCOUNT_ID=843587364827107",
		"META_GRAPH_BASE_URL=https://graph.facebook.com",
		"META_GRAPH_VERSION=v26.0",
		"META_ADS_CACHE_TTL_SEC=86400",
		"META_ADS_REQUEST_TIMEOUT_SEC=30",
		"META_ADS_MAX_RANGE_DAYS=366",
		"META_LEAD_ACTION_TYPES=lead",
		"META_ATTRIBUTION_WINDOWS=",
		"",
	}, "\n")

	if err := os.WriteFile(os.Args[1], []byte(contents), 0o400); err != nil {
		fatal(err)
	}
	fmt.Println("Backend-only Meta Ads environment created.")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
