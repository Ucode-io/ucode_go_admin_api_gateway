package v1

import (
	"net/http/httptest"
	"reflect"
	"strings"
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

func TestCRMRequestRecognizesScreenshotCardMatchAsFieldSettings(t *testing.T) {
	message := "lidlarimni cardini shunaqa qiber"
	if !crmRequestLooksLikeFieldSettings(message) {
		t.Fatalf("expected %q to be recognized as a field-setting request", message)
	}
	request := models.CRMAssistantRequest{
		Message:     message,
		Images:      []string{"data:image/jpeg;base64,eA=="},
		PageContext: models.CRMAssistantPageContext{Table: "deals"},
	}
	if crmRequestRequiresLiveLookup(request) {
		t.Fatal("screenshot card matching must not be misclassified as a live analytics lookup")
	}
}

func TestCRMMutationCancelMessageUsesRequestLanguage(t *testing.T) {
	tests := map[string]string{
		"Ksjdd deal budgetini o‘zgartir": "Amal bekor qilindi. Hech narsa o‘zgarmadi.",
		"Update the Ksjdd deal budget":   "Action cancelled. No data was changed.",
		"Измени бюджет сделки Ksjdd":     "Действие отменено. Данные не изменены.",
	}
	for message, want := range tests {
		if got := crmMutationCancelMessage(message); got != want {
			t.Fatalf("message %q: got %q, want %q", message, got, want)
		}
	}
}

func TestDetectCRMRequestLanguageHandlesMixedCRMVocabulary(t *testing.T) {
	tests := map[string]string{
		"Ksjdd deal budgetini o‘zgartir": "uz",
		"Update the Ksjdd deal budget":   "en",
		"Измени бюджет сделки Ksjdd":     "ru",
		"kecha kelgan лидlar":            "uz",
	}
	for message, want := range tests {
		if got := detectCRMRequestLanguage(message); got != want {
			t.Fatalf("message %q: got %q, want %q", message, got, want)
		}
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

func TestBuildCommonCRMFieldSettingsShowHide(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
			{Slug: "amount", Label: "Сумма"},
		},
	}}
	result, ok := buildCommonCRMFieldSettings(models.CRMAssistantRequest{
		Message:     "source korinadigon qiber keyin budgetni yashir kartochkada",
		PageContext: models.CRMAssistantPageContext{Table: "deals"},
	}, schema)
	if !ok || len(result.clientActions) != 1 {
		t.Fatalf("expected field action, got %#v, ok=%v", result, ok)
	}
	action := result.clientActions[0]
	if !reflect.DeepEqual(action.ShowFields, []string{"source"}) || !reflect.DeepEqual(action.HideFields, []string{"amount"}) {
		t.Fatalf("unexpected show/hide action: %#v", action)
	}
}

func TestBuildCommonCRMFieldSettingsUnderstandsNegativeShowForm(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
			{Slug: "amount", Label: "Сумма"},
		},
	}}
	result, ok := buildCommonCRMFieldSettings(models.CRMAssistantRequest{
		Message:     "kartochkada source korinsin budgetni korsatma",
		PageContext: models.CRMAssistantPageContext{Table: "deals"},
	}, schema)
	if !ok || len(result.clientActions) != 1 {
		t.Fatalf("expected field action, got %#v, ok=%v", result, ok)
	}
	action := result.clientActions[0]
	if !reflect.DeepEqual(action.ShowFields, []string{"source"}) || !reflect.DeepEqual(action.HideFields, []string{"amount"}) {
		t.Fatalf("unexpected negative-show action: %#v", action)
	}
}

func TestBuildCommonCRMFieldSettingsOrdersPhoneAndSource(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "source", Label: "Source"},
		},
	}}
	result, ok := buildCommonCRMFieldSettings(models.CRMAssistantRequest{
		Message:     "kartochkada telefonni birinchi source ni ikkinchi qiber",
		PageContext: models.CRMAssistantPageContext{Table: "deals"},
	}, schema)
	if !ok || len(result.clientActions) != 1 {
		t.Fatalf("expected ordered field action, got %#v, ok=%v", result, ok)
	}
	action := result.clientActions[0]
	want := []string{"contacts_id", "source"}
	if !reflect.DeepEqual(action.FieldOrder, want) || !reflect.DeepEqual(action.ShowFields, want) {
		t.Fatalf("unexpected ordered field action: %#v", action)
	}
}

func TestBuildCommonCRMFieldSettingsLeavesScreenshotsForVisionAgent(t *testing.T) {
	result, ok := buildCommonCRMFieldSettings(models.CRMAssistantRequest{
		Message:     "source ni ko‘rsat",
		Images:      []string{"data:image/png;base64,eA=="},
		PageContext: models.CRMAssistantPageContext{Table: "deals"},
	}, []models.TableSchema{{Slug: "deals", Fields: []models.FieldSchema{{Slug: "source"}}}})
	if ok || result != nil {
		t.Fatal("screenshot-backed settings must be handled by the vision agent")
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

func TestFormatSchemaForCRMIncludesLabelsAndStatusOptions(t *testing.T) {
	formatted := formatSchemaForCRM([]models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "pipeline", Label: "Voronka", Type: "MULTISELECT"},
			{Slug: "pipeline_sales", Label: "Sales Project", Type: "STATUS", Options: []string{"Yangi lid", "Sotildi"}},
		},
	}})

	for _, expected := range []string{
		`pipeline MULTISELECT [label="Voronka"]`,
		`pipeline_sales STATUS [label="Sales Project"] [options="Yangi lid|Sotildi"]`,
	} {
		if !strings.Contains(formatted, expected) {
			t.Fatalf("formatted schema %q does not contain %q", formatted, expected)
		}
	}
}

func TestCanonicalDealScreenshotCardActionUsesActiveStatusAndHidesOtherFields(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "name", Type: "SINGLE_LINE"},
			{Slug: "pipeline", Type: "MULTISELECT"},
			{Slug: "amount", Type: "NUMBER"},
			{Slug: "pipeline_sales_project", Type: "STATUS"},
			{Slug: "pipeline_other", Type: "STATUS"},
			{Slug: "contacts_id", Type: "LOOKUP"},
			{Slug: "notes", Type: "MULTI_LINE"},
		},
	}}
	pageContext := models.CRMAssistantPageContext{
		Table: "deals",
		CardFields: []models.CRMCardFieldContext{
			{Slug: "pipeline"},
			{Slug: "name"},
			{Slug: "contacts_id"},
			{Slug: "amount"},
			{Slug: "pipeline_sales_project"},
			{Slug: "notes"},
		},
	}

	got, ok := canonicalDealScreenshotCardAction(schema, pageContext)
	if !ok {
		t.Fatal("expected the active deal status field to produce a canonical screenshot action")
	}
	want := models.CRMClientAction{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"name", "pipeline", "amount", "pipeline_sales_project"},
		HideFields: []string{"contacts_id", "notes"},
		FieldOrder: []string{"name", "pipeline", "amount", "pipeline_sales_project"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical screenshot action = %#v, want %#v", got, want)
	}
}

func TestCanonicalDealScreenshotCardActionRejectsAmbiguousActiveStatus(t *testing.T) {
	schema := []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "pipeline_a", Type: "STATUS"},
			{Slug: "pipeline_b", Type: "STATUS"},
		},
	}}
	pageContext := models.CRMAssistantPageContext{
		Table: "deals",
		CardFields: []models.CRMCardFieldContext{
			{Slug: "name"},
			{Slug: "pipeline"},
			{Slug: "amount"},
			{Slug: "pipeline_a"},
			{Slug: "pipeline_b"},
		},
	}

	if _, ok := canonicalDealScreenshotCardAction(schema, pageContext); ok {
		t.Fatal("ambiguous active STATUS fields must not be guessed")
	}
}

func TestCRMClientActionsContainCommonDealScreenshotCore(t *testing.T) {
	actions := []models.CRMClientAction{{
		Type:       "set_card_field_visibility",
		Table:      "deals",
		ShowFields: []string{"pipeline", "contacts_id", "amount"},
		FieldOrder: []string{"pipeline", "name", "contacts_id", "amount"},
	}}
	if !crmClientActionsContainFields(actions, "name", "pipeline", "amount") {
		t.Fatal("expected the model's partially-correct screenshot mapping to be recognized")
	}
	if crmClientActionsContainFields(actions, "name", "pipeline", "amount", "pipeline_sales_project") {
		t.Fatal("missing active status must not be reported as present")
	}
}

func TestExtractSchemaFieldOptionsFindsNestedStatusGroups(t *testing.T) {
	attributes := map[string]any{
		"todo": map[string]any{
			"options": []any{
				map[string]any{"label_en": "Yangi lid", "value": "new"},
				map[string]any{"label_en": "Primary contact", "value": "primary"},
			},
		},
		"complete": map[string]any{
			"options": []any{
				map[string]any{"label_en": "Sotildi", "value": "won"},
			},
		},
	}

	want := map[string]bool{"Yangi lid": true, "Primary contact": true, "Sotildi": true}
	got := extractSchemaFieldOptions(attributes)
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %d unique options", got, len(want))
	}
	for _, option := range got {
		if !want[option] {
			t.Fatalf("unexpected extracted option %q from %#v", option, got)
		}
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
