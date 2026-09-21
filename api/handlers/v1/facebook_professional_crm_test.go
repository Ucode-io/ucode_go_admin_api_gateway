package v1

import (
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestDefaultCRMMappingUsesUdevsPipeline(t *testing.T) {
	mapping := defaultCRMMapping()

	if mapping.PipelineValue != "Udevs" {
		t.Fatalf("PipelineValue = %q, want Udevs", mapping.PipelineValue)
	}
	if mapping.PipelineStageField != "pipeline_udevs" {
		t.Fatalf("PipelineStageField = %q, want pipeline_udevs", mapping.PipelineStageField)
	}
}

func TestProfessionalCRMLeadFieldsReadsLocalizedNameQuestion(t *testing.T) {
	fields := professionalCRMLeadFields([]models.FacebookFieldData{
		{Name: "ismingiz_nima", Values: []string{"Ali Valiyev"}},
		{Name: "phone_number", Values: []string{"+998901234567"}},
	})

	if fields.fullName() != "Ali Valiyev" {
		t.Fatalf("fullName() = %q, want Ali Valiyev", fields.fullName())
	}
}

func TestProfessionalCRMLeadFieldsDoesNotUseCompanyName(t *testing.T) {
	fields := professionalCRMLeadFields([]models.FacebookFieldData{
		{Name: "company_name", Values: []string{"Udevs"}},
		{Name: "phone_number", Values: []string{"+998901234567"}},
	})

	if fields.fullName() != "" {
		t.Fatalf("fullName() = %q, want blank", fields.fullName())
	}
}

func TestCRMEnableNewLeadForm(t *testing.T) {
	payload := map[string]any{}

	crmEnableNewLeadForm(payload, "accept_leads")

	accepted, ok := payload["accept_leads"].(bool)
	if !ok || !accepted {
		t.Fatalf("accept_leads = %#v, want true", payload["accept_leads"])
	}
}

func TestCRMEnableNewLeadFormWithoutConfiguredField(t *testing.T) {
	payload := map[string]any{}

	crmEnableNewLeadForm(payload, "")

	if len(payload) != 0 {
		t.Fatalf("payload = %#v, want unchanged payload", payload)
	}
}

func TestCRMLeadFormEnableUpdateRepairsDisabledRow(t *testing.T) {
	update := crmLeadFormEnableUpdate(map[string]any{
		"guid":         "form-guid",
		"accept_leads": false,
	}, "accept_leads")

	if update == nil {
		t.Fatal("expected update payload")
	}
	if update["guid"] != "form-guid" || update["accept_leads"] != true {
		t.Fatalf("unexpected update payload: %#v", update)
	}
}

func TestCRMLeadFormEnableUpdateSkipsEnabledRow(t *testing.T) {
	update := crmLeadFormEnableUpdate(map[string]any{
		"guid":         "form-guid",
		"accept_leads": true,
	}, "accept_leads")

	if update != nil {
		t.Fatalf("update = %#v, want nil", update)
	}
}
