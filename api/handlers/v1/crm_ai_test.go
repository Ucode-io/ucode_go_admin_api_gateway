package v1

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"

	"github.com/gin-gonic/gin"
)

func TestCRMRequestRequiresLiveLookupUnderstandsInformalLanguageAndFollowUps(t *testing.T) {
	tests := []struct {
		name    string
		request models.CRMAssistantRequest
		want    bool
	}{
		{
			name:    "informal phone request",
			request: models.CRMAssistantRequest{Message: "kechigi kegan lidlani nomerini tashavor"},
			want:    true,
		},
		{
			name:    "client synonym",
			request: models.CRMAssistantRequest{Message: "kecha tushgan klientlarni aloqa raqamlari kerak"},
			want:    true,
		},
		{
			name: "short follow up",
			request: models.CRMAssistantRequest{
				Message: "nomerlarini ham ber",
				History: []models.CRMAssistantMessage{{Role: "user", Content: "kecha kelgan lidlar qaysi statusda"}},
			},
			want: true,
		},
		{
			name:    "analytical superlative",
			request: models.CRMAssistantRequest{Message: "eng katta budgetli deal qaysi"},
			want:    true,
		},
		{
			name:    "field settings are not data lookup",
			request: models.CRMAssistantRequest{Message: "lid kartochkasida source ko‘rinadigan qil"},
			want:    false,
		},
		{
			name:    "ordinary conversation",
			request: models.CRMAssistantRequest{Message: "rahmat, zo‘r bo‘ldi"},
			want:    false,
		},
		{
			name: "implicit current page entity",
			request: models.CRMAssistantRequest{
				Message:     "kechagi oqimdan kelganlani telini ber",
				PageContext: models.CRMAssistantPageContext{Table: "deals"},
			},
			want: true,
		},
		{
			name: "field settings without saying card",
			request: models.CRMAssistantRequest{
				Message:     "source korinadigon qiber budgetni yashir",
				PageContext: models.CRMAssistantPageContext{Table: "deals"},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := crmRequestRequiresLiveLookup(test.request); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCRMReplyIsClarification(t *testing.T) {
	if !crmReplyIsClarification("Qaysi davrni nazarda tutdingiz?") {
		t.Fatal("expected clarification question")
	}
	if crmReplyIsClarification("Kechikkan lidlar topilmadi.") {
		t.Fatal("a guessed no-results answer must not bypass the live lookup guard")
	}
}

func TestCRMLeadPeriodQueryRequiresCreatedAt(t *testing.T) {
	tests := []struct {
		message string
		want    bool
	}{
		{message: "avgustda qaysi source eng ko‘p lid olib kelgan", want: true},
		{message: "o‘tgan oygi lid oqimida qaysi kun pik bo‘lgan", want: true},
		{message: "shu oygi leadlar umumiy summasi", want: true},
		{message: "eng katta budgetli deal qaysi", want: false},
		{message: "avgustdagi lidlarni start_date bo‘yicha guruhla", want: false},
	}

	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			request := models.CRMAssistantRequest{Message: test.message}
			if got := crmLeadPeriodQueryRequiresCreatedAt(request); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCRMSQLUsesCreatedAt(t *testing.T) {
	if !crmSQLUsesCreatedAt(`SELECT COUNT(*) FROM "deals" WHERE d."created_at" >= $1`) {
		t.Fatal("created_at query was not recognized")
	}
	if crmSQLUsesCreatedAt(`SELECT COUNT(*) FROM "deals" WHERE "start_date" >= $1`) {
		t.Fatal("start_date must not satisfy the creation-time guard")
	}
	if crmSQLUsesCreatedAt(`SELECT "created_at" FROM "deals" WHERE "start_date" >= $1`) {
		t.Fatal("selecting created_at must not bypass a start_date filter")
	}
}

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

func TestNormalizeCRMClientActionsResolvesDealPhoneToContactField(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
		},
	}}
	actions := normalizeCRMClientActions([]models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"phone"},
		FieldOrder: []string{"source", "phone"},
	}}, schema, "deals")

	want := []models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"contacts_id"},
		HideFields: []string{},
		FieldOrder: []string{"source", "contacts_id"},
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
