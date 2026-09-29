package v1

import (
	"testing"

	"github.com/google/uuid"
)

func TestMissedCallIdentityUsesCallNotPhone(t *testing.T) {
	first := missedCallDealID("project-1", "call-1")
	if first != missedCallDealID("project-1", "call-1") {
		t.Fatal("replayed call must have the same deal id")
	}
	if first == missedCallDealID("project-1", "call-2") {
		t.Fatal("a new call from the same phone must create a new deal")
	}
	if first == missedCallDealID("project-2", "call-1") {
		t.Fatal("different CRM projects must not share a deal")
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Fatalf("deal id must be a UUID: %v", err)
	}
}

func TestMissedCallFromRow(t *testing.T) {
	base := map[string]any{
		"call_uuid":    "call-1",
		"client_phone": "+998 90 123 45 67",
		"started_at":   "2026-09-29T09:30:00Z",
		"direction":    "incoming",
		"status":       "no_answer",
	}
	if id, phone, _, ok := missedCallFromRow(base); !ok || id != "call-1" || phone != "+998 90 123 45 67" {
		t.Fatalf("valid missed incoming call was rejected: id=%q phone=%q ok=%v", id, phone, ok)
	}
	for _, change := range []map[string]any{
		{"direction": "outgoing"},
		{"status": "answered"},
		{"status": "initiated"},
		{"call_uuid": ""},
		{"client_phone": ""},
	} {
		row := make(map[string]any, len(base))
		for key, value := range base {
			row[key] = value
		}
		for key, value := range change {
			row[key] = value
		}
		if _, _, _, ok := missedCallFromRow(row); ok {
			t.Fatalf("non-missed call accepted: %+v", change)
		}
	}
}
