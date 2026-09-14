package v1

import "testing"

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
