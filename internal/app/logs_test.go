package app

import (
	"testing"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestNormalizeLogBody(t *testing.T) {
	body := normalizeLogBody("[INFO] first line\n[warn] second line\nthird line")
	if body != "first line\nsecond line\nthird line" {
		t.Fatalf("normalized body = %q", body)
	}
}

func TestLiveLogDeduperSuppressesRecentHistoryOverlap(t *testing.T) {
	deduper := newLiveLogDeduper([]client.Log{
		{Level: "INFO", Data: "old"},
		{Level: "INFO", Data: "repeat"},
	})

	if !deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "[INFO] repeat"}) {
		t.Fatal("expected first overlapping message to be suppressed")
	}
	if deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "new"}) {
		t.Fatal("expected first new live message to be printed")
	}
	if deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "repeat"}) {
		t.Fatal("expected deduper to disable after new live message")
	}
}

func TestLiveLogDeduperHandlesDuplicateHistoryLines(t *testing.T) {
	deduper := newLiveLogDeduper([]client.Log{
		{Level: "INFO", Data: "same"},
		{Level: "INFO", Data: "same"},
	})

	if !deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "same"}) {
		t.Fatal("expected first duplicate overlap to be suppressed")
	}
	if !deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "[INFO] same"}) {
		t.Fatal("expected second duplicate overlap to be suppressed")
	}
	if deduper.ShouldSuppress(client.LogStreamMessage{Level: "INFO", Data: "same"}) {
		t.Fatal("expected further duplicate after handoff to be printed")
	}
}

func TestFormatStreamingLogsNoHistory(t *testing.T) {
	formatted := formatStreamingLogs(nil, 0, true)
	if formatted != "Following live output for this resource. Waiting for logs..." {
		t.Fatalf("formatted = %q", formatted)
	}
}

func TestFormatRecentLogsNoHistory(t *testing.T) {
	formatted := formatRecentLogs(nil, 0, true)
	if formatted != "Waiting for logs..." {
		t.Fatalf("formatted = %q", formatted)
	}
}

func TestFormattedMapListSection(t *testing.T) {
	formatted := formattedMapListSection([]map[string]any{{"env": "prod", "team": "platform"}, {"region": "eu-west-1"}})
	if formatted != "1. env=prod, team=platform\n2. region=eu-west-1" {
		t.Fatalf("formatted section = %q", formatted)
	}
}
