package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Ensure(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return nil
}

func EventsPath(dir string) string { return filepath.Join(dir, "events.jsonl") }
func CursorPath(dir string) string { return filepath.Join(dir, "cursor.json") }
func LockPath(dir string) string   { return filepath.Join(dir, "store.lock") }

func StableID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func AppendRecords(dir string, records []EvidenceRecord) error {
	if len(records) == 0 {
		return nil
	}
	if err := Ensure(dir); err != nil {
		return err
	}
	f, err := os.OpenFile(EventsPath(dir), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, rec := range records {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return f.Sync()
}

func LoadRecords(dir string) ([]EvidenceRecord, error) {
	f, err := os.Open(EventsPath(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []EvidenceRecord
	seen := map[string]bool{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec EvidenceRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		if rec.ID != "" && seen[rec.ID] {
			continue
		}
		seen[rec.ID] = true
		records = append(records, rec)
	}
	return records, scanner.Err()
}

func Status(dir string) (StatusInfo, error) {
	st := StatusInfo{StoreDir: dir, EventsPath: EventsPath(dir), CursorPath: CursorPath(dir), CheckedAt: time.Now()}
	if info, err := os.Stat(EventsPath(dir)); err == nil {
		st.StoreBytes = info.Size()
	}
	records, err := LoadRecords(dir)
	if err != nil {
		return st, err
	}
	st.Events = len(records)
	sessions := map[string]bool{}
	for _, rec := range records {
		if rec.SessionID != "" {
			sessions[rec.SessionID] = true
		}
		if rec.IngestedAt != "" && rec.IngestedAt > st.LastIngested {
			st.LastIngested = rec.IngestedAt
		}
	}
	st.Sessions = len(sessions)
	cursor, err := LoadCursor(dir)
	if err == nil {
		st.Transcripts = len(cursor.Transcripts)
	}
	return st, nil
}

func WriteStatus(w io.Writer, st StatusInfo, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}
	fmt.Fprintf(w, "agent-recall status\n\n")
	fmt.Fprintf(w, "Store:       %s\n", st.StoreDir)
	fmt.Fprintf(w, "Events:      %d\n", st.Events)
	fmt.Fprintf(w, "Sessions:    %d\n", st.Sessions)
	fmt.Fprintf(w, "Transcripts: %d\n", st.Transcripts)
	fmt.Fprintf(w, "Bytes:       %d\n", st.StoreBytes)
	if st.LastIngested != "" {
		fmt.Fprintf(w, "Last sync:   %s\n", st.LastIngested)
	}
	return nil
}
