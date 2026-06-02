package models

import (
	"regexp"
	"strings"
)

// appKeywordRe matches explicit "application product" wording that should map to a
// webapp (an end-user mobile app), e.g. "web app", "webapp", "mobile app".
var appKeywordRe = regexp.MustCompile(`(?i)\b(web ?app|webapp|mobile ?app|mobile application|web application)\b`)

// appForRe matches product phrasing like "app for tracking ..." / "app to manage ...".
var appForRe = regexp.MustCompile(`(?i)\bapp\s+(for|to)\b`)

// marketingSignals indicate the user explicitly wants a marketing site / promo page,
// in which case we must NOT override to webapp even if the word "app" appears
// (e.g. "build a landing page for my app").
var marketingSignals = []string{
	"landing page", "landing-page",
	"promo page", "promotional page", "promo site",
	"marketing site", "marketing website", "website for",
	"one-page", "one page site", "single page site", "single-page site",
	"coming soon",
}

// adminSignals indicate the user EXPLICITLY wants an internal back-office / admin tool.
// When present we keep "admin_panel" even if app-wording also appears
// (e.g. "an app for our staff to manage orders — an admin panel").
var adminSignals = []string{
	"admin panel", "admin-panel", "admin console", "admin dashboard",
	"back office", "back-office", "backoffice",
	"management system", "management dashboard", "management panel",
	"control panel", "internal tool", "internal dashboard",
	"staff portal", "for staff", "for our staff", "for administrators",
	"erp", "cms",
}

// ApplyProjectTypeKeywordOverride deterministically corrects the architect's
// project-type classification for the most common miss: the user explicitly asked
// for an "app" / "web app" / "mobile app" (a phone product) but the LLM, swayed by a
// marketing-flavored or management-flavored description, classified it as "web",
// "landing", or "admin_panel".
//
// Behavior:
//   - Leaves an already-correct "webapp" untouched.
//   - Backs off entirely when the prompt has explicit marketing-site signals
//     (so "build a landing page for my app" stays a landing page).
//   - When app-wording is present, upgrades "web"/"landing" → "webapp", and ALSO
//     "admin_panel" → "webapp" UNLESS the user explicitly asked for an admin/
//     back-office tool (adminSignals).
//
// There is no native/mobile project type, so "mobile app" is intentionally mapped to
// "webapp" (a mobile-styled responsive web app).
func ApplyProjectTypeKeywordOverride(plan *ArchitectPlan, userPrompt string) {
	if plan == nil || plan.ProjectType == "webapp" {
		return
	}

	lower := strings.ToLower(userPrompt)

	// Respect explicit marketing/promo intent — do not override these.
	for _, s := range marketingSignals {
		if strings.Contains(lower, s) {
			return
		}
	}

	// Only act on explicit app-product wording.
	if !appKeywordRe.MatchString(lower) && !appForRe.MatchString(lower) {
		return
	}

	// For an admin classification, respect an EXPLICIT admin/back-office request.
	if plan.ProjectType == "admin_panel" {
		for _, s := range adminSignals {
			if strings.Contains(lower, s) {
				return // user really wants an admin panel — keep it
			}
		}
	}

	// web, landing, or non-explicit admin_panel + app wording → webapp (mobile app).
	plan.ProjectType = "webapp"
}
