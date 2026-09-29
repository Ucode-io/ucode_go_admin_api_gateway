package v1

import (
	"encoding/json"
	"testing"

	"ucode/ucode_go_api_gateway/genproto/company_service"
)

func TestProjectWithDescriptionKeepsProjectFields(t *testing.T) {
	response := projectWithDescription{
		Project: &company_service.Project{
			ProjectId: "project-id",
			Title:     "Professional CRM",
			Logo:      "bucket/Media/logo.png",
		},
		Description: "CRM workspace",
	}
	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"project_id":  "project-id",
		"title":       "Professional CRM",
		"logo":        "bucket/Media/logo.png",
		"description": "CRM workspace",
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %s", key, got[key], want)
		}
	}
}
