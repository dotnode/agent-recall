package search

import (
	"testing"
	"time"

	"agent-recall/internal/store"
)

func TestRecallScoresFiltersAndLimits(t *testing.T) {
	dir := t.TempDir()
	appendSearchRecords(t, dir, []store.EvidenceRecord{
		searchRecord("old", "s1", "/repo", "assistant", "message", "auth callback", 10),
		searchRecord("new", "s1", "/repo", "user", "decision", "auth callback decided", 30),
		searchRecord("other-session", "s2", "/repo", "user", "decision", "auth callback", 40),
		searchRecord("other-kind", "s1", "/repo", "user", "tool_use", "auth callback", 50),
	})

	res, err := Recall(dir, Query{Text: "auth callback", Limit: 1, SessionID: "s1", CWD: "/repo", Kind: "decision"})
	if err != nil {
		t.Fatalf("Recall() error = %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "new" {
		t.Fatalf("items = %+v", res.Items)
	}
	if res.Items[0].Score <= 0 {
		t.Fatalf("score = %v, want positive", res.Items[0].Score)
	}
}

func TestRecallTieBreaksByNewestByteStart(t *testing.T) {
	dir := t.TempDir()
	appendSearchRecords(t, dir, []store.EvidenceRecord{
		searchRecord("first", "s1", "/repo", "assistant", "message", "auth", 10),
		searchRecord("second", "s1", "/repo", "assistant", "message", "auth", 20),
	})

	res, err := Recall(dir, Query{Text: "auth"})
	if err != nil {
		t.Fatalf("Recall() error = %v", err)
	}
	if len(res.Items) != 2 || res.Items[0].ID != "second" || res.Items[1].ID != "first" {
		t.Fatalf("items = %+v", res.Items)
	}
}

func TestTimelineReturnsRecentRecords(t *testing.T) {
	dir := t.TempDir()
	appendSearchRecords(t, dir, []store.EvidenceRecord{
		searchRecord("one", "s1", "/repo", "assistant", "message", "first", 1),
		searchRecord("two", "s1", "/repo", "assistant", "message", "second", 2),
		searchRecord("other", "s2", "/repo", "assistant", "message", "other", 3),
	})

	res, err := Timeline(dir, Query{SessionID: "s1", Limit: 2})
	if err != nil {
		t.Fatalf("Timeline() error = %v", err)
	}
	if len(res.Items) != 2 || res.Items[0].ID != "two" || res.Items[1].ID != "one" {
		t.Fatalf("items = %+v", res.Items)
	}
}

func TestDecisionsFiltersDecisionKind(t *testing.T) {
	dir := t.TempDir()
	appendSearchRecords(t, dir, []store.EvidenceRecord{
		searchRecord("message", "s1", "/repo", "user", "message", "decided auth", 1),
		searchRecord("decision", "s1", "/repo", "user", "decision", "decided auth", 2),
	})

	res, err := Decisions(dir, Query{Text: "auth"})
	if err != nil {
		t.Fatalf("Decisions() error = %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "decision" {
		t.Fatalf("items = %+v", res.Items)
	}
}

func appendSearchRecords(t *testing.T, dir string, records []store.EvidenceRecord) {
	t.Helper()
	if err := store.AppendRecords(dir, records); err != nil {
		t.Fatalf("AppendRecords() error = %v", err)
	}
}

func searchRecord(id, sessionID, cwd, role, kind, text string, byteStart int64) store.EvidenceRecord {
	return store.EvidenceRecord{
		Version:    1,
		ID:         id,
		Agent:      "claude-code",
		SessionID:  sessionID,
		CWD:        cwd,
		Timestamp:  "2026-05-16T00:00:00Z",
		IngestedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Role:       role,
		Kind:       kind,
		Text:       text,
		Source:     store.EvidenceSource{Type: "transcript", TranscriptPath: "session.jsonl", Line: byteStart, ByteStart: byteStart},
	}
}
