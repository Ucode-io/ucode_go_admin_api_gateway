package callquality

import (
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestCalculateResultValidatesScoresEvidenceAndPenaltyCaps(t *testing.T) {
	req := models.CallQualityEvaluateRequest{
		CallID: "call-1", TranscriptSource: "stereo", CriteriaVersion: "v1",
		Transcript: []models.CallQualityTranscriptLine{{Speaker: "operator", Text: "Assalomu alaykum, gaplashishga qulaymi?", From: 1200}},
		Criteria: []models.CallQualityCriterion{
			{ID: "opening", Name: "Opening", MaxScore: 10, AllowedScores: []float64{0, 5, 10}},
			{ID: "qualification", Name: "Qualification", MaxScore: 15, AllowedScores: []float64{0, 5, 10, 15}},
		},
	}
	raw := rawEvaluation{
		Criteria: []models.CallQualityCriterionResult{
			{CriterionID: "opening", Score: 10, Status: statusCompleted, Evidence: []models.CallQualityEvidence{{Speaker: "operator", Quote: "gaplashishga qulaymi?"}}},
			{CriterionID: "qualification", Score: 12, Status: statusCompleted},
		},
		Violations: []models.CallQualityViolation{
			{CriterionID: "opening", Severity: severityModerate, Evidence: []models.CallQualityEvidence{{Speaker: "operator", Quote: "Assalomu alaykum"}}},
			{CriterionID: "opening", Severity: severityMajor, Evidence: []models.CallQualityEvidence{{Speaker: "operator", Quote: "qulaymi?"}}},
			{CriterionID: "opening", Severity: severityMajor, Evidence: []models.CallQualityEvidence{{Speaker: "operator", Quote: "topilmagan"}}},
		},
	}

	got := calculateResult(req, raw)
	if got.PositiveScore != 10 || got.Penalty != -2 || got.TotalScore != 8 || got.MaxScore != 25 {
		t.Fatalf("unexpected totals: %+v", got)
	}
	if !got.ReviewRequired || got.Criteria[1].Status != statusUncertain {
		t.Fatalf("invalid score must require review: %+v", got.Criteria[1])
	}
	if len(got.Violations) != 2 || got.Violations[1].Penalty != -1 {
		t.Fatalf("per-block penalty cap was not applied: %+v", got.Violations)
	}
	if got.Criteria[0].Evidence[0].From != 1200 {
		t.Fatalf("evidence timestamp must come from transcript")
	}
}

func TestCalculateResultMarksMonoTranscriptForReview(t *testing.T) {
	req := models.CallQualityEvaluateRequest{
		CallID: "call-2", TranscriptSource: "mono", CriteriaVersion: "v1",
		Transcript: []models.CallQualityTranscriptLine{{Speaker: "unknown", Text: "Salom"}},
		Criteria:   []models.CallQualityCriterion{{ID: "opening", Name: "Opening", MaxScore: 10, AllowedScores: []float64{10}}},
	}
	got := calculateResult(req, rawEvaluation{Criteria: []models.CallQualityCriterionResult{{CriterionID: "opening", Score: 10, Status: statusProtected}}})
	if !got.ReviewRequired || len(got.ReviewReasons) == 0 {
		t.Fatalf("mono transcript must require review: %+v", got)
	}
}

func TestValidateRequestRejectsDuplicateCriteria(t *testing.T) {
	req := models.CallQualityEvaluateRequest{
		Transcript: []models.CallQualityTranscriptLine{{Speaker: "operator", Text: "Salom"}},
		Criteria: []models.CallQualityCriterion{
			{ID: "opening", Name: "Opening", Instructions: "Check greeting", MaxScore: 10, AllowedScores: []float64{0, 10}},
			{ID: "opening", Name: "Duplicate", Instructions: "Check duplicate", MaxScore: 5, AllowedScores: []float64{0, 5}},
		},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected duplicate criterion validation error")
	}
}
