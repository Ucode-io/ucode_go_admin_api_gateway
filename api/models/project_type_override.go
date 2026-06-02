package models

import (
	"regexp"
	"strings"
)

// appKeywordRe matches explicit "application product" wording that should map to a
// webapp (an end-user product workspace), e.g. "web app", "webapp", "mobile app".
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

// ApplyProjectTypeKeywordOverride deterministically corrects the architect's
// project-type classification for the most common miss: the user explicitly asked
// for an "app" / "web app" / "mobile app" (a product workspace) but the LLM, swayed
// by a marketing-flavored description, classified it as "web" or "landing".
//
// It is intentionally conservative:
//   - It only ever upgrades "web"/"landing" → "webapp". It never touches "admin_panel"
//     or an already-correct "webapp".
//   - It backs off entirely when the prompt contains explicit marketing-site signals,
//     so "build a landing page for my app" stays a landing page.
//
// There is no native/mobile project type, so "mobile app" is intentionally mapped to
// "webapp" (a responsive web application).
func ApplyProjectTypeKeywordOverride(plan *ArchitectPlan, userPrompt string) {
	if plan == nil {
		return
	}
	// Only rescue marketing classifications; leave admin_panel and webapp untouched.
	if plan.ProjectType != "web" && plan.ProjectType != "landing" {
		return
	}

	lower := strings.ToLower(userPrompt)

	// Respect explicit marketing/promo intent — do not override these.
	for _, s := range marketingSignals {
		if strings.Contains(lower, s) {
			return
		}
	}

	if appKeywordRe.MatchString(lower) || appForRe.MatchString(lower) {
		plan.ProjectType = "webapp"
	}
}
