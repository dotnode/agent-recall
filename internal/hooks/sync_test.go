package hooks

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-recall/internal/store"
)

func TestSyncIngestsRedactsAndUpdatesCursor(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTranscript(t, []string{
		`{"type":"user","timestamp":"2026-05-16T10:00:00Z","uuid":"u1","sessionId":"s1","message":{"role":"user","content":"token sk-ant-secret123 should be hidden"}}`,
		`{"type":"assistant","timestamp":"2026-05-16T10:01:00Z","uuid":"a1","sessionId":"s1","message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`,
	})

	if err := Sync(hookInput(t, transcript), SyncOptions{StoreDir: dir}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	records, err := store.LoadRecords(dir)
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %+v", records)
	}
	if !records[0].Redaction.Applied || strings.Contains(records[0].Text, "sk-ant-secret123") {
		t.Fatalf("redaction = %+v text=%q", records[0].Redaction, records[0].Text)
	}

	cursor, err := store.LoadCursor(dir)
	if err != nil {
		t.Fatalf("LoadCursor() error = %v", err)
	}
	key := store.CursorKey(transcript)
	if cursor.Transcripts[key].Line != 2 || cursor.Transcripts[key].Offset == 0 {
		t.Fatalf("cursor = %+v", cursor.Transcripts[key])
	}

	if err := Sync(hookInput(t, transcript), SyncOptions{StoreDir: dir}); err != nil {
		t.Fatalf("Sync() second error = %v", err)
	}
	records, err = store.LoadRecords(dir)
	if err != nil {
		t.Fatalf("LoadRecords() second error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records after second sync = %+v", records)
	}
}

func TestSyncFlushDerivesDecisionRecords(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTranscript(t, []string{
		`{"type":"user","timestamp":"2026-05-16T10:00:00Z","uuid":"u1","sessionId":"s1","message":{"role":"user","content":"WE SHOULD keep the installer small."}}`,
		`{"type":"assistant","timestamp":"2026-05-16T10:01:00Z","uuid":"a1","sessionId":"s1","message":{"role":"assistant","content":"普通消息"}}`,
	})

	if err := Sync(hookInput(t, transcript), SyncOptions{StoreDir: dir, Flush: true}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	records, err := store.LoadRecords(dir)
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	var decisions int
	for _, rec := range records {
		if rec.Kind == "decision" {
			decisions++
			if rec.Meta["matched_keyword"] != "we should" {
				t.Fatalf("decision meta = %+v", rec.Meta)
			}
		}
	}
	if decisions != 1 {
		t.Fatalf("decisions = %d records=%+v", decisions, records)
	}
}

func TestSyncReportsMalformedTranscriptLine(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTranscript(t, []string{
		`not-json`,
		`{"type":"user","timestamp":"2026-05-16T10:00:00Z","uuid":"u1","sessionId":"s1","message":{"role":"user","content":"valid text"}}`,
	})

	report, err := SyncWithReport(hookInput(t, transcript), SyncOptions{StoreDir: dir})
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("SyncWithReport() error = %v", err)
	}
	if report.ParseErrors != 1 || report.RecordsWritten != 1 {
		t.Fatalf("report = %+v", report)
	}
	records, err := store.LoadRecords(dir)
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].Text != "valid text" {
		t.Fatalf("records = %+v", records)
	}
}

func TestSyncReportsNoExtractedText(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTranscript(t, []string{
		`{"type":"summary","timestamp":"2026-05-16T10:00:00Z","uuid":"u1","sessionId":"s1"}`,
	})

	report, err := SyncWithReport(hookInput(t, transcript), SyncOptions{StoreDir: dir})
	if err == nil || !strings.Contains(err.Error(), "extracted no text") {
		t.Fatalf("SyncWithReport() error = %v", err)
	}
	if report.EmptyTextSkipped != 1 || report.RecordsWritten != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestContainsFoldHandlesUnicodeCase(t *testing.T) {
	if !containsFold("DÉCIDED", "décided") {
		t.Fatalf("expected Unicode case-insensitive match")
	}
	if !containsFold("采用这个方案", "方案") {
		t.Fatalf("expected Chinese substring match")
	}
}

func hookInput(t *testing.T, transcript string) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"hook_event_name":   "Stop",
		"session_id":        "s1",
		"transcript_path":   transcript,
		"cwd":               "/tmp/project",
		"custom_unused_key": true,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return bytes.NewReader(b)
}

func writeTranscript(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
