package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

func TestExtractReferenceURLsNormalizesBareDomain(t *testing.T) {
	urls := extractReferenceURLs("Make landing page according to this thesecrettrading.de website. Make 1 to 1 logic and design.")

	if len(urls) != 1 {
		t.Fatalf("expected one URL, got %d: %#v", len(urls), urls)
	}
	if urls[0] != "https://thesecrettrading.de" {
		t.Fatalf("unexpected URL: %s", urls[0])
	}
}

func TestNormalizeReferenceURLBlocksPrivateHosts(t *testing.T) {
	tests := []string{
		"http://localhost:3000",
		"http://127.0.0.1",
		"http://10.0.0.5",
		"http://172.16.0.1",
		"http://192.168.1.4",
		"http://service-name",
		"file:///etc/passwd",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if normalized, err := normalizeReferenceURL(raw); err == nil {
				t.Fatalf("expected %q to be blocked, got %s", raw, normalized)
			}
		})
	}
}

func TestPrepareReferencePromptCapturesAndAppendsScreenshots(t *testing.T) {
	var gotRequest referenceCaptureRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(models.ReferenceSiteContext{
			URL:         gotRequest.URL,
			FinalURL:    "https://thesecrettrading.de/",
			Title:       "The Secret Trading",
			Description: "Trading landing page",
			Screenshots: []models.ReferenceScreenshot{
				{Viewport: "desktop", URL: "https://cdn.example.com/desktop.png", Width: 1440, Height: 1200},
				{Viewport: "mobile", URL: "https://cdn.example.com/mobile.png", Width: 390, Height: 1200},
			},
			Colors:   []string{"#050505", "#f4c542"},
			Fonts:    []string{"Inter"},
			Sections: []models.ReferenceSection{{Heading: "Hero", Copy: "Learn trading", CTA: "Start"}},
		})
	}))
	defer srv.Close()

	prompt, images, ref, msg := prepareReferencePrompt(
		context.Background(),
		config.BaseConfig{
			ReferenceCaptureEnabled:        true,
			ReferenceCaptureURL:            srv.URL,
			ReferenceCaptureTimeoutSeconds: 5,
		},
		"Make landing page according to this thesecrettrading.de website. Make 1 to 1 logic and design.",
		[]string{"https://cdn.example.com/existing.png"},
	)

	if msg != "" {
		t.Fatalf("expected no user message, got %q", msg)
	}
	if ref == nil {
		t.Fatal("expected reference context")
	}
	if gotRequest.URL != "https://thesecrettrading.de" {
		t.Fatalf("capture called with wrong URL: %s", gotRequest.URL)
	}
	if !strings.Contains(prompt, referenceContextMarker) {
		t.Fatalf("enriched prompt missing reference marker:\n%s", prompt)
	}
	if len(images) != 3 {
		t.Fatalf("expected existing + 2 screenshots, got %#v", images)
	}
}

func TestPrepareReferencePromptFallsBackToHTMLExtraction(t *testing.T) {
	oldFetch := fetchReferenceSiteHTMLForPrompt
	defer func() { fetchReferenceSiteHTMLForPrompt = oldFetch }()

	fetchReferenceSiteHTMLForPrompt = func(_ context.Context, targetURL string) (*models.ReferenceSiteContext, error) {
		return &models.ReferenceSiteContext{
			URL:         targetURL,
			Title:       "Example",
			Description: "Reference extracted from HTML",
			Colors:      []string{"#111111"},
			Fonts:       []string{"Inter"},
			Sections:    []models.ReferenceSection{{Heading: "Hero", Copy: "Welcome", CTA: "Start"}},
		}, nil
	}

	prompt, images, ref, msg := prepareReferencePrompt(
		context.Background(),
		config.BaseConfig{},
		"Clone https://example.com 1:1",
		nil,
	)

	if ref != nil {
		if ref.Title != "Example" {
			t.Fatalf("unexpected ref title: %s", ref.Title)
		}
	} else {
		t.Fatal("expected HTML reference context")
	}
	if msg != "" {
		t.Fatalf("expected no clarification, got %q", msg)
	}
	if len(images) != 0 {
		t.Fatalf("HTML fallback should not append screenshots, got %#v", images)
	}
	if !strings.Contains(prompt, "HTML/CSS-only extraction") {
		t.Fatalf("expected HTML/CSS warning in prompt:\n%s", prompt)
	}
}

func TestExtractReferenceSiteFromHTMLPullsStyleAndContent(t *testing.T) {
	htmlText := `
		<!doctype html>
		<html>
			<head>
				<title>Secret Trading</title>
				<meta name="description" content="Premium trading education">
				<style>
					body { color: #f5f5f5; background: #050505; font-family: "Inter", sans-serif; }
				</style>
			</head>
			<body>
				<header class="hero"><h1>Master the Markets</h1><p>Learn private trading logic.</p><a href="/start">Join now</a></header>
				<img src="/hero.png" alt="Trading dashboard">
			</body>
		</html>`
	cssText := `@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@700&display=swap'); .btn { color: #f4c542; }`

	ref := extractReferenceSiteFromHTML("https://thesecrettrading.de", "https://thesecrettrading.de", htmlText, cssText)

	if ref.Title != "Secret Trading" {
		t.Fatalf("unexpected title: %s", ref.Title)
	}
	if len(ref.Sections) == 0 || ref.Sections[0].Heading != "Master the Markets" {
		t.Fatalf("expected hero section, got %#v", ref.Sections)
	}
	if len(ref.Assets) == 0 || ref.Assets[0].URL != "https://thesecrettrading.de/hero.png" {
		t.Fatalf("expected resolved image asset, got %#v", ref.Assets)
	}
	if !containsString(ref.Colors, "#050505") || !containsString(ref.Colors, "#f4c542") {
		t.Fatalf("expected extracted colors, got %#v", ref.Colors)
	}
	if !containsString(ref.Fonts, "Inter") || !containsString(ref.Fonts, "Space Grotesk") {
		t.Fatalf("expected extracted fonts, got %#v", ref.Fonts)
	}
}

func TestApplyReferenceContextToPlanSuppressesTablesForPureClone(t *testing.T) {
	ref := &models.ReferenceSiteContext{
		URL: "https://example.com",
		Screenshots: []models.ReferenceScreenshot{
			{Viewport: "desktop", URL: "https://cdn.example.com/desktop.png"},
		},
	}
	plan := &models.ArchitectPlan{
		ProjectType: "landing",
		Tables: []models.TablePlan{
			{Slug: "leads", Label: "Leads"},
		},
		Relations: []models.TableRelationPlan{
			{TableFrom: "leads", TableTo: "users", Type: "Many2One"},
		},
		ClientTypes: []string{"Administrator"},
	}

	applyReferenceContextToPlan(plan, ref, "Make landing page according to example.com. Make 1 to 1 design.")

	if !plan.CloneMode || plan.Reference == nil {
		t.Fatal("expected clone mode and reference context")
	}
	if len(plan.Tables) != 0 || len(plan.Relations) != 0 || len(plan.ClientTypes) != 0 {
		t.Fatalf("expected pure clone to suppress backend data, got tables=%d relations=%d clientTypes=%d", len(plan.Tables), len(plan.Relations), len(plan.ClientTypes))
	}
	if !strings.Contains(plan.UIStructure, referenceContextMarker) {
		t.Fatal("expected UI structure to include reference context")
	}
}

func TestApplyReferenceContextToPlanKeepsTablesForDynamicClone(t *testing.T) {
	ref := &models.ReferenceSiteContext{
		URL: "https://example.com",
		Screenshots: []models.ReferenceScreenshot{
			{Viewport: "desktop", URL: "https://cdn.example.com/desktop.png"},
		},
	}
	plan := &models.ArchitectPlan{
		ProjectType: "admin_panel",
		Tables: []models.TablePlan{
			{Slug: "orders", Label: "Orders"},
		},
	}

	applyReferenceContextToPlan(plan, ref, "Create an admin panel like example.com for managing orders")

	if len(plan.Tables) != 1 {
		t.Fatalf("expected dynamic clone to keep tables, got %d", len(plan.Tables))
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
