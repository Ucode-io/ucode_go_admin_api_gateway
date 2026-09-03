package v1

import (
	"reflect"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestNormalizeCRMClientActionsResolvesLabelsAndConflicts(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
			{Slug: "budget", Label: "Deal budget"},
			{Slug: "phone_number", Label: "Phone number"},
		},
	}}
	actions := normalizeCRMClientActions([]models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		ShowFields: []string{"Source", "phone number", "source"},
		HideFields: []string{"Deal budget", "source", "unknown"},
		FieldOrder: []string{"Phone number", "Source"},
	}}, schema, "deals")

	want := []models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"source", "phone_number"},
		HideFields: []string{"budget"},
		FieldOrder: []string{"phone_number", "source"},
	}}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("unexpected actions:\n got: %#v\nwant: %#v", actions, want)
	}
}

func TestValidateCRMImagesAcceptsInlineScreenshot(t *testing.T) {
	if err := validateCRMImages([]string{"data:image/png;base64,eA=="}); err != nil {
		t.Fatalf("valid screenshot rejected: %v", err)
	}
}

func TestValidateCRMImagesRejectsRemoteURL(t *testing.T) {
	if err := validateCRMImages([]string{"https://example.com/screenshot.png"}); err == nil {
		t.Fatal("remote screenshot URL must be rejected")
	}
}

func TestNormalizePreferenceFields(t *testing.T) {
	got := normalizePreferenceFields([]string{"source", "source", "bad field", "budget"})
	want := []string{"source", "budget"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
