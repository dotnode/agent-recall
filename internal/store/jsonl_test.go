package store

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStableID(t *testing.T) {
	a := StableID("agent", "session", "line")
	b := StableID("agent", "session", "line")
	c := StableID("agent", "session", "other")
	if a != b {
		t.Fatalf("stable id changed")
	}
	if a == c {
		t.Fatalf("stable id collision for different inputs")
	}
}

func TestAppendLoadRecordsDedupesAndCountsBadLinesInStatus(t *testing.T) {
	dir := t.TempDir()
	rec := EvidenceRecord{
		Version:    1,
		ID:         "ev-1",
		Agent:      "claude-code",
		SessionID:  "s1",
		IngestedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Role:       "user",
		Kind:       "message",
		Text:       "auth context",
		Source:     EvidenceSource{Type: "transcript", TranscriptPath: "session.jsonl", Line: 1},
	}
	if err := AppendRecords(dir, []EvidenceRecord{rec, rec}); err != nil {
		t.Fatalf("AppendRecords() error = %v", err)
	}
	f, err := os.OpenFile(EventsPath(dir), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := f.WriteString("not-json\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	records, err := LoadRecords(dir)
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].ID != "ev-1" {
		t.Fatalf("records = %+v", records)
	}

	st, err := Status(dir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if st.Events != 1 || st.BadLines != 1 || st.Sessions != 1 {
		t.Fatalf("status = %+v", st)
	}
}

func TestStatusReportsCursorError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(CursorPath(dir), []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	st, err := Status(dir)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if st.CursorError == "" {
		t.Fatalf("CursorError = empty, status = %+v", st)
	}

	var text bytes.Buffer
	if err := WriteStatus(&text, st, false); err != nil {
		t.Fatalf("WriteStatus() error = %v", err)
	}
	if !strings.Contains(text.String(), "Cursor error:") {
		t.Fatalf("text status = %q", text.String())
	}
}

func TestWriteStatusReportsBadLines(t *testing.T) {
	st := StatusInfo{StoreDir: "/tmp/store", Events: 1, BadLines: 2, CheckedAt: time.Now()}
	var text bytes.Buffer
	if err := WriteStatus(&text, st, false); err != nil {
		t.Fatalf("WriteStatus() text error = %v", err)
	}
	if !strings.Contains(text.String(), "Bad lines:   2") {
		t.Fatalf("text status = %q", text.String())
	}

	var jsonOut bytes.Buffer
	if err := WriteStatus(&jsonOut, st, true); err != nil {
		t.Fatalf("WriteStatus() json error = %v", err)
	}
	var decoded StatusInfo
	if err := json.Unmarshal(jsonOut.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.BadLines != 2 {
		t.Fatalf("decoded = %+v", decoded)
	}
}
