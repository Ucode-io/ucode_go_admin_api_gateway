package v1

import "testing"

func TestApplyGoogleDriveRedirectPlaceholdersSupportsMcpProjectID(t *testing.T) {
	target := "https://ugenn.netlify.app/projects/{mcp_project_id}?tab=dashboard&section=resources"
	state := googleDriveOAuthState{
		ProjectID:     "37fecd3e-dde6-4714-9691-08f1970d6d2f",
		EnvironmentID: "1971aabb-5682-4ae9-a49a-58cf72c10a76",
		McpProjectID:  "58be0b19-371f-4ff6-af41-50878b12743c",
	}

	got := applyGoogleDriveRedirectPlaceholders(target, state)
	want := "https://ugenn.netlify.app/projects/58be0b19-371f-4ff6-af41-50878b12743c?tab=dashboard&section=resources"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
