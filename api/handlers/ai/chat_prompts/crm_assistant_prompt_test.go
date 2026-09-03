package chat_prompts

import (
	"strings"
	"testing"
)

func TestBuildRelativeDateHintResolvesUzbekYesterdayInCRMTimezone(t *testing.T) {
	hint := buildRelativeDateHint(
		"kecha nechta lid keldi",
		"2026-09-03T04:45:00Z",
		"Asia/Tashkent",
	)

	for _, expected := range []string{
		"requested_period=yesterday",
		"local_date=2026-09-02",
		"[2026-09-02T00:00:00+05:00, 2026-09-03T00:00:00+05:00)",
	} {
		if !strings.Contains(hint, expected) {
			t.Fatalf("relative date hint %q does not contain %q", hint, expected)
		}
	}
}

func TestBuildRelativeDateHintResolvesRussianToday(t *testing.T) {
	hint := buildRelativeDateHint(
		"Сколько лидов пришло сегодня?",
		"2026-09-03T04:45:00Z",
		"Asia/Tashkent",
	)

	if !strings.Contains(hint, "requested_period=today") || !strings.Contains(hint, "local_date=2026-09-03") {
		t.Fatalf("unexpected relative date hint: %q", hint)
	}
}

func TestBuildRelativeDateHintIgnoresAbsoluteQuestion(t *testing.T) {
	if hint := buildRelativeDateHint("Sentabr oyida nechta lid keldi?", "2026-09-03T04:45:00Z", "Asia/Tashkent"); hint != "" {
		t.Fatalf("unexpected relative date hint: %q", hint)
	}
}
