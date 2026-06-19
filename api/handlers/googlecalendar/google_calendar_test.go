package googlecalendar

import (
	"testing"

	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

func TestEventFromDataMapsProjectStatusToGoogleStatus(t *testing.T) {
	event, err := eventFromData(testMappingWithStatus(), map[string]any{
		"title":      "New Meeting",
		"start_date": "2026-06-20 11:52:00",
		"end_date":   "2026-06-20 12:53:00",
		"status":     "scheduled",
	})
	if err != nil {
		t.Fatalf("eventFromData() error = %v", err)
	}
	if event.Status != "confirmed" {
		t.Fatalf("event.Status = %q, want %q", event.Status, "confirmed")
	}
}

func TestEventFromDataIgnoresUnknownProjectStatus(t *testing.T) {
	event, err := eventFromData(testMappingWithStatus(), map[string]any{
		"title":      "New Meeting",
		"start_date": "2026-06-20 11:52:00",
		"end_date":   "2026-06-20 12:53:00",
		"status":     "in-review",
	})
	if err != nil {
		t.Fatalf("eventFromData() error = %v", err)
	}
	if event.Status != "" {
		t.Fatalf("event.Status = %q, want empty status", event.Status)
	}
}

func testMappingWithStatus() *pb.GoogleCalendarMapping {
	return &pb.GoogleCalendarMapping{
		TableSlug:   "meetings",
		TitleField:  "title",
		StartField:  "start_date",
		EndField:    "end_date",
		StatusField: "status",
	}
}
