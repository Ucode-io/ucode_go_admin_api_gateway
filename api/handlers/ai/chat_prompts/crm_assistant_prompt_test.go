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
		"local_wall_interval=[2026-09-02 00:00:00, 2026-09-03 00:00:00)",
		"offset_interval=[2026-09-02T00:00:00+05:00, 2026-09-03T00:00:00+05:00)",
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

func TestBuildRelativeDateHintResolvesWeekAndMonthRanges(t *testing.T) {
	tests := []struct {
		message  string
		period   string
		interval string
	}{
		{message: "bu hafta nechta lid keldi", period: "this_week", interval: "[2026-08-31 00:00:00, 2026-09-07 00:00:00)"},
		{message: "oldingi hafta qancha lid kegan", period: "last_week", interval: "[2026-08-24 00:00:00, 2026-08-31 00:00:00)"},
		{message: "otkan hafta qancha lid kegan", period: "last_week", interval: "[2026-08-24 00:00:00, 2026-08-31 00:00:00)"},
		{message: "bu oy nechta lid keldi", period: "this_month", interval: "[2026-09-01 00:00:00, 2026-10-01 00:00:00)"},
		{message: "o'tgan oy nechta lid keldi", period: "last_month", interval: "[2026-08-01 00:00:00, 2026-09-01 00:00:00)"},
		{message: "otkan oy qancha lid kegan", period: "last_month", interval: "[2026-08-01 00:00:00, 2026-09-01 00:00:00)"},
	}
	for _, test := range tests {
		hint := buildRelativeDateHint(test.message, "2026-09-03T04:45:00Z", "Asia/Tashkent")
		if !strings.Contains(hint, "requested_period="+test.period) || !strings.Contains(hint, "local_wall_interval="+test.interval) {
			t.Fatalf("unexpected hint for %q: %q", test.message, hint)
		}
	}
}
