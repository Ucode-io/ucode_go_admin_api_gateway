package v1

import (
	"strings"
	"testing"
)

func TestCapacitorPromptAddendumKeepsWebAppFlow(t *testing.T) {
	config := capacitorPromptAddendum()

	if strings.Contains(config, "React Native + Expo") {
		t.Fatalf("Capacitor prompt must not request React Native generation:\n%s", config)
	}
	for _, expected := range []string{"React + Vite", "Capacitor", "src/lib/capacitor.ts", "NEVER emit ios/ or android/"} {
		if !strings.Contains(config, expected) {
			t.Fatalf("expected Capacitor prompt to contain %q", expected)
		}
	}
}
