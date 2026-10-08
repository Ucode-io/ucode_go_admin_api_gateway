package config

import "testing"

func TestCRMNativeReviewOptIn(t *testing.T) {
	for _, value := range []string{"", "false", "TRUE", "1", "true"} {
		t.Setenv("CRM_NATIVE_REVIEW_ENABLED", value)
		if BaseLoad().CRMNativeReviewEnabled != (value == "true") {
			t.Fatalf("reader activation for %q", value)
		}
	}
}
