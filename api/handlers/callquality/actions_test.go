package callquality

import (
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestValidateActionsRequiresTranscriptEvidence(t *testing.T) {
	req := actionRequest{
		StartedAt: "2026-09-29T09:00:00+05:00",
		Transcript: []models.CallQualityTranscriptLine{
			{Speaker: "operator", Text: "Yoshingiz nechida?"},
			{Speaker: "client", Text: "Men 25 yoshdaman. Chorshanba kuni uchrashamiz."},
		},
	}
	result := validateActions(req, actionResult{
		Age: &extractedAge{Value: 25, Quote: "Men 25 yoshdaman"},
		Tasks: []extractedTask{
			{Title: "Mijoz bilan uchrashish", Type: "meeting", DueDate: "2026-09-30", Quote: "Chorshanba kuni uchrashamiz"},
			{Title: "Invented", Type: "call", DueDate: "2026-09-30", Quote: "not in transcript"},
		},
	})
	if result.Age == nil || result.Age.Value != 25 || len(result.Tasks) != 1 {
		t.Fatalf("unexpected validated actions: %+v", result)
	}

	result = validateActions(req, actionResult{Age: &extractedAge{Value: 25, Quote: "Yoshingiz nechida?"}})
	if result.Age != nil {
		t.Fatal("operator's question must not be accepted as client's age")
	}
}
