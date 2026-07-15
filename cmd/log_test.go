package cmd

import (
	"testing"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
)

func TestNormalizeStreamBody(t *testing.T) {
	body := normalizeStreamBody("[INFO] first line\n[warn] second line\nthird line")
	if body != "first line\nsecond line\nthird line" {
		t.Fatalf("normalized body = %q", body)
	}
}

func TestFormatHistoricalLogNoColors(t *testing.T) {
	formatted := formatHistoricalLog(client.Log{Level: "INFO", Data: "[INFO] hello"}, true)
	if formatted != "hello" {
		t.Fatalf("formatted log = %q", formatted)
	}
}

func TestLogDeduperSuppressesRecentHistoryOverlap(t *testing.T) {
	deduper := newLogDeduper([]client.Log{
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

func TestLogDeduperHandlesDuplicateHistoryLines(t *testing.T) {
	deduper := newLogDeduper([]client.Log{
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

func TestParseSinceDuration(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	since, err := parseSince("90m", now)
	if err != nil {
		t.Fatalf("parse since duration: %v", err)
	}
	if since == nil {
		t.Fatal("expected since timestamp")
	}
	want := now.Add(-90 * time.Minute)
	if !since.Equal(want) {
		t.Fatalf("since = %s, want %s", since.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestParseSinceRFC3339(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	since, err := parseSince("2026-07-02T10:30:00Z", now)
	if err != nil {
		t.Fatalf("parse since timestamp: %v", err)
	}
	if since == nil || since.Format(time.RFC3339) != "2026-07-02T10:30:00Z" {
		t.Fatalf("unexpected since = %v", since)
	}
}

func TestParseSinceInvalid(t *testing.T) {
	if _, err := parseSince("yesterdayish", time.Now()); err == nil {
		t.Fatal("expected invalid since error")
	}
}

func TestFilterLogsSince(t *testing.T) {
	since := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	logs := []client.Log{
		{ID: "1", CreatedAt: time.Date(2026, 7, 2, 9, 59, 59, 0, time.UTC)},
		{ID: "2", CreatedAt: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)},
		{ID: "3", CreatedAt: time.Date(2026, 7, 2, 10, 0, 1, 0, time.UTC)},
	}

	filtered := filterLogsSince(logs, &since)
	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d", len(filtered))
	}
	if filtered[0].ID != "2" || filtered[1].ID != "3" {
		t.Fatalf("filtered ids = %#v", []string{filtered[0].ID, filtered[1].ID})
	}
}

func TestLogStartMessage(t *testing.T) {
	resourceItem := client.Resource{ID: "r1", Name: "redis-prod"}

	message := logStartMessage("resource", resourceItem.Name, resourceItem.ID, nil, false)
	if message != "Showing recent logs for resource redis-prod (r1)." {
		t.Fatalf("message = %q", message)
	}

	since := time.Date(2026, 7, 2, 10, 30, 0, 0, time.UTC)
	message = logStartMessage("resource", resourceItem.Name, resourceItem.ID, &since, true)
	if message != "Showing logs since 2026-07-02T10:30:00Z for resource redis-prod (r1). Following live output until Ctrl-C." {
		t.Fatalf("message = %q", message)
	}
}

func TestLogStartMessageExecutor(t *testing.T) {
	message := logStartMessage("executor", "terraform-runner", "e1", nil, true)
	if message != "Showing recent logs for executor terraform-runner (e1). Following live output until Ctrl-C." {
		t.Fatalf("message = %q", message)
	}
}

func TestLogEntityMeta(t *testing.T) {
	entityID, name, label, err := logEntityMeta("executor", client.Executor{ID: "e1", Name: "terraform-runner"})
	if err != nil {
		t.Fatalf("logEntityMeta returned error: %v", err)
	}
	if entityID != "e1" || name != "terraform-runner" || label != "executor" {
		t.Fatalf("unexpected meta = %q %q %q", entityID, name, label)
	}
}
