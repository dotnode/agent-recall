package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

func CursorKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	h := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(h[:])
}

func LoadCursor(dir string) (CursorState, error) {
	var state CursorState
	state.Version = 1
	state.Transcripts = map[string]TranscriptCursor{}
	b, err := os.ReadFile(CursorPath(dir))
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return state, err
	}
	if state.Transcripts == nil {
		state.Transcripts = map[string]TranscriptCursor{}
	}
	return state, nil
}

func SaveCursor(dir string, state CursorState) error {
	if err := Ensure(dir); err != nil {
		return err
	}
	state.Version = 1
	if state.Transcripts == nil {
		state.Transcripts = map[string]TranscriptCursor{}
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := CursorPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, CursorPath(dir))
}

func UpdatedCursor(key, path, sessionID string, offset, line, size int64, modTime time.Time, lastRecordID string) TranscriptCursor {
	return TranscriptCursor{
		Key:          key,
		Path:         path,
		SessionID:    sessionID,
		Offset:       offset,
		Line:         line,
		Size:         size,
		ModTime:      modTime.UTC().Format(time.RFC3339Nano),
		LastRecordID: lastRecordID,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
}
