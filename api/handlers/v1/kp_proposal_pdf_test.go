package v1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsValidKpRequestID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid", id: "KP-20260804-D6BE339096AF", want: true},
		{name: "valid with underscore", id: "KP-2026_08_04-abc", want: true},
		{name: "empty", id: "", want: false},
		{name: "missing prefix", id: "20260804-D6BE339096AF", want: false},
		{name: "prefix only, nothing after", id: "KP-", want: false},
		{name: "lowercase prefix rejected", id: "kp-20260804-abc", want: false},
		{name: "contains slash", id: "KP-2026/08/04", want: false},
		{name: "contains space", id: "KP-2026 08 04", want: false},
		{name: "contains path traversal", id: "KP-../../etc/passwd", want: false},
		{name: "over max length", id: "KP-" + strings.Repeat("a", 94), want: false},
		{name: "at max length", id: "KP-" + strings.Repeat("a", 93), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidKpRequestID(tc.id); got != tc.want {
				t.Fatalf("isValidKpRequestID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

// TestKpPdfCacheEntryRoundTrip guards the wire format shared between
// GenerateKpProposal (writer) and DownloadKpProposalPDF (reader): a field tag
// typo on either side would silently break tenant isolation instead of
// failing loudly.
func TestKpPdfCacheEntryRoundTrip(t *testing.T) {
	want := kpPdfCacheEntry{
		ProjectID:     "37fecd3e-dde6-4714-9691-08f1970d6d2f",
		EnvironmentID: "1971aabb-5682-4ae9-a49a-58cf72c10a76",
		Title:         "SaaS-платформа",
	}

	body, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got kpPdfCacheEntry
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got != want {
		t.Fatalf("round trip mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}
