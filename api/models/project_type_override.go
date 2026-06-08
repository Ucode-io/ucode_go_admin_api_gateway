package models

import (
	"regexp"
	"strings"
)

// appKeywordRe matches EXPLICIT consumer/end-user app wording that should map to a
// webapp (an end-user mobile app), e.g. "web app", "webapp", "mobile app".
// This is the STRONG signal — strong enough to override even an admin_panel guess.
var appKeywordRe = regexp.MustCompile(`(?i)\b(web ?app|webapp|mobile ?app|mobile application|web application)\b`)

// appForRe matches product phrasing like "app for tracking ..." / "app to manage ...".
// This is a WEAK, ambiguous signal: "an app to manage orders" is almost always an admin
// tool, not a consumer app. So appForRe only upgrades web/landing → webapp and is
// NEVER allowed to downgrade an admin_panel into a sidebar-less mobile app.
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

// managementIntentRe catches business-data MANAGEMENT framing ("manage orders",
// "inventory", "warehouse", "back office", CRUD) that means an admin_panel even when the
// user also says "app". A management tool rendered as a sidebar-less mobile webapp is the
// worst regression, so this guard protects an admin_panel classification from being
// downgraded. ("manag\w*" covers manage/managing/management/manager.)
var managementIntentRe = regexp.MustCompile(`(?i)\b(manag\w*|inventory|warehouse|back[\s-]?office|crud)\b`)

// ApplyProjectTypeKeywordOverride deterministically corrects the architect's
// project-type classification for the most common miss: the user explicitly asked
// for a consumer "app" / "web app" / "mobile app" (a phone product) but the LLM, swayed
// by a marketing-flavored description, classified it as "web" or "landing".
//
// Behavior:
//   - Leaves an already-correct "webapp" untouched.
//   - Backs off entirely on explicit marketing-site signals
//     (so "build a landing page for my app" stays a landing page).
//   - web / landing → webapp on ANY app-wording (explicit "web app" OR weak "app for/to").
//   - admin_panel → webapp ONLY on EXPLICIT consumer-app wording ("mobile app"/"web app"),
//     never on the weak "app for/to" phrasing, and only when there is NO admin or
//     management framing. This keeps real admin/management tools as admin panels
//     (with their sidebar) instead of turning them into sidebar-less mobile apps.
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

	hasExplicitApp := appKeywordRe.MatchString(lower)
	hasAppForTo := appForRe.MatchString(lower)
	if !hasExplicitApp && !hasAppForTo {
		return
	}

	switch plan.ProjectType {
	case "web", "landing":
		// Core rescue: a consumer app misclassified as a website. Any app-wording upgrades it.
		plan.ProjectType = "webapp"
	case "admin_panel":
		// Conservative: the architect detected internal/management intent. Only override on
		// EXPLICIT consumer-app wording, never the weak "app for/to", and only when there is
		// no admin/management framing at all.
		if !hasExplicitApp {
			return
		}
		for _, s := range adminSignals {
			if strings.Contains(lower, s) {
				return // user really wants an admin panel — keep it (with its sidebar)
			}
		}
		if managementIntentRe.MatchString(lower) {
			return // "manage X / inventory / back office" → admin panel, not a mobile app
		}
		plan.ProjectType = "webapp"
	}
}
