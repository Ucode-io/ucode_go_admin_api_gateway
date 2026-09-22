package v2

import "testing"

func TestNormalizeProfessionalCRMLegacyDeal(t *testing.T) {
	data := map[string]any{
		"pipeline":               legacyProfessionalCRMPipeline,
		"stage":                  "Новая заявка",
		"pipeline_sales_project": "Новая заявка",
	}

	normalizeProfessionalCRMLegacyDeal(professionalCRMProjectID, professionalCRMDealsCollection, data)

	if data["pipeline"] != professionalCRMPipeline {
		t.Fatalf("pipeline = %#v, want %q", data["pipeline"], professionalCRMPipeline)
	}
	if data[professionalCRMStageField] != "Новая заявка" {
		t.Fatalf("%s = %#v, want %q", professionalCRMStageField, data[professionalCRMStageField], "Новая заявка")
	}
}

func TestNormalizeProfessionalCRMLegacyDealSupportsJSONArrays(t *testing.T) {
	data := map[string]any{
		"pipeline": []any{legacyProfessionalCRMPipeline},
		"stage":    []any{"Новая заявка"},
	}

	normalizeProfessionalCRMLegacyDeal(professionalCRMProjectID, professionalCRMDealsCollection, data)

	pipeline, ok := data["pipeline"].([]any)
	if !ok || len(pipeline) != 1 || pipeline[0] != professionalCRMPipeline {
		t.Fatalf("pipeline = %#v, want [%q]", data["pipeline"], professionalCRMPipeline)
	}
	if data[professionalCRMStageField] != "Новая заявка" {
		t.Fatalf("%s = %#v, want %q", professionalCRMStageField, data[professionalCRMStageField], "Новая заявка")
	}
}

func TestNormalizeProfessionalCRMLegacyDealIsProjectScoped(t *testing.T) {
	data := map[string]any{"pipeline": legacyProfessionalCRMPipeline}

	normalizeProfessionalCRMLegacyDeal("11111111-1111-1111-1111-111111111111", professionalCRMDealsCollection, data)

	if data["pipeline"] != legacyProfessionalCRMPipeline {
		t.Fatalf("pipeline = %#v, want unchanged", data["pipeline"])
	}
	if _, ok := data[professionalCRMStageField]; ok {
		t.Fatalf("unexpected %s in payload", professionalCRMStageField)
	}
}

func TestNormalizeProfessionalCRMLegacyDealLeavesCurrentPipelineUntouched(t *testing.T) {
	data := map[string]any{
		"pipeline":               professionalCRMPipeline,
		"pipeline_udevs":         "Первичный контакт",
		"pipeline_sales_project": "Новая заявка",
	}

	normalizeProfessionalCRMLegacyDeal(professionalCRMProjectID, professionalCRMDealsCollection, data)

	if data[professionalCRMStageField] != "Первичный контакт" {
		t.Fatalf("%s = %#v, want unchanged", professionalCRMStageField, data[professionalCRMStageField])
	}
}

func TestIsTelegramDealStageUpdate(t *testing.T) {
	if !isTelegramDealStageUpdate(map[string]any{"pipeline_uhrms": "Выграно"}) {
		t.Fatal("pipeline-specific stage update should be recognized")
	}
	if isTelegramDealStageUpdate(map[string]any{"name": "Changed deal"}) {
		t.Fatal("unrelated update should not be treated as a stage update")
	}
}
