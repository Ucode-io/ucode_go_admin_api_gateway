package v1

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"

	"github.com/gin-gonic/gin"
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

func TestNormalizeCRMClientActionsResolvesBudgetAliasToAmount(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
			{Slug: "amount", Label: "Сумма"},
		},
	}}
	actions := normalizeCRMClientActions([]models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"source"},
		HideFields: []string{"budget"},
	}}, schema, "deals")

	want := []models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"source"},
		HideFields: []string{"amount"},
		FieldOrder: []string{},
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

func TestGetAiChatUserIDUsesCRMContextWithoutWritingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("user_id", "crm-user-id")

	userID, err := (&HandlerV1{}).getAiChatUserID(context)
	if err != nil {
		t.Fatalf("getAiChatUserID returned an error: %v", err)
	}
	if userID != "crm-user-id" {
		t.Fatalf("got user ID %q, want %q", userID, "crm-user-id")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("identity lookup unexpectedly wrote an HTTP response: %s", recorder.Body.String())
	}
}

func TestGetAiChatUserIDUsesClientAuthFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("Auth", &auth.V2HasAccessUserRes{UserIdAuth: "auth-user-id"})

	userID, err := (&HandlerV1{}).getAiChatUserID(context)
	if err != nil {
		t.Fatalf("getAiChatUserID returned an error: %v", err)
	}
	if userID != "auth-user-id" {
		t.Fatalf("got user ID %q, want %q", userID, "auth-user-id")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("identity lookup unexpectedly wrote an HTTP response: %s", recorder.Body.String())
	}
}
