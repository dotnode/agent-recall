package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agent-recall/internal/config"
	"agent-recall/internal/redact"
	"agent-recall/internal/store"
	"agent-recall/internal/transcript"
)

type SyncOptions struct {
	StoreDir string
	Flush    bool
}

type SyncReport struct {
	LinesRead        int
	RecordsWritten   int
	ParseErrors      int
	EmptyTextSkipped int
	DecisionsDerived int
}

func (r SyncReport) Warning() string {
	switch {
	case r.ParseErrors > 0:
		return fmt.Sprintf("failed to parse %d transcript line(s); transcript schema or file contents may be invalid", r.ParseErrors)
	case r.LinesRead > 0 && r.RecordsWritten == 0:
		return fmt.Sprintf("read %d transcript line(s) but extracted no text; transcript schema may have changed", r.LinesRead)
	default:
		return ""
	}
}

func Sync(stdin io.Reader, opts SyncOptions) error {
	_, err := SyncWithReport(stdin, opts)
	return err
}

func SyncWithReport(stdin io.Reader, opts SyncOptions) (SyncReport, error) {
	input, err := ReadInput(stdin)
	if err != nil {
		return SyncReport{}, err
	}
	dir, err := config.StoreDir(opts.StoreDir)
	if err != nil {
		return SyncReport{}, err
	}
	lock, err := store.AcquireLock(dir)
	if err != nil {
		return SyncReport{}, err
	}
	defer lock.Release()

	cursorState, err := store.LoadCursor(dir)
	if err != nil {
		return SyncReport{}, err
	}
	key := store.CursorKey(input.TranscriptPath)
	cursor := cursorState.Transcripts[key]
	result, err := transcript.ReadNew(input.TranscriptPath, cursor.Offset, cursor.Line)
	if err != nil {
		return SyncReport{}, err
	}

	redactor := redact.Default()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var records []store.EvidenceRecord
	lastID := cursor.LastRecordID
	report := SyncReport{LinesRead: len(result.Lines)}
	for _, line := range result.Lines {
		parsed, err := transcript.Parse(line.Raw)
		if err != nil {
			report.ParseErrors++
			continue
		}
		if parsed.Text == "" {
			report.EmptyTextSkipped++
			continue
		}
		sessionID := input.SessionID
		if sessionID == "" {
			sessionID = parsed.SessionID
		}
		redacted := redactor.Redact(parsed.Text)
		rawHash := sha256.Sum256(line.Raw)
		id := store.StableID("claude-code", sessionID, input.TranscriptPath, fmt.Sprint(line.Number), fmt.Sprint(line.ByteStart), hex.EncodeToString(rawHash[:]))
		rec := store.EvidenceRecord{
			Version:    1,
			ID:         id,
			Agent:      "claude-code",
			SessionID:  sessionID,
			CWD:        input.CWD,
			Timestamp:  parsed.Timestamp,
			IngestedAt: now,
			Role:       parsed.Role,
			Kind:       normalizeKind(parsed.Kind),
			Text:       redacted.Text,
			Source: store.EvidenceSource{
				Type:           "transcript",
				TranscriptPath: input.TranscriptPath,
				Line:           line.Number,
				ByteStart:      line.ByteStart,
				ByteEnd:        line.ByteEnd,
				HookEventName:  input.HookEventName,
				TranscriptUUID: parsed.UUID,
			},
			Redaction: store.RedactionInfo{Applied: redacted.Applied, Rules: redacted.Rules},
		}
		records = append(records, rec)
		report.RecordsWritten++
		lastID = id
	}

	if opts.Flush {
		decisionSource := records
		if len(decisionSource) == 0 {
			if existing, loadErr := store.LoadRecords(dir); loadErr == nil {
				decisionSource = existing
			}
		}
		decisions := deriveDecisions(decisionSource, now)
		report.DecisionsDerived = len(decisions)
		records = append(records, decisions...)
	}
	if err := store.AppendRecords(dir, records); err != nil {
		return report, err
	}
	info, statErr := os.Stat(input.TranscriptPath)
	var size int64 = result.Size
	var modTime = result.ModTime
	if statErr == nil {
		size = info.Size()
		modTime = info.ModTime()
	}
	cursorState.Transcripts[key] = store.UpdatedCursor(key, input.TranscriptPath, input.SessionID, result.Offset, result.Line, size, modTime, lastID)
	if err := store.SaveCursor(dir, cursorState); err != nil {
		return report, err
	}
	if warning := report.Warning(); warning != "" {
		return report, fmt.Errorf("%s", warning)
	}
	return report, nil
}

func normalizeKind(kind string) string {
	switch kind {
	case "tool_use", "tool_result", "decision", "marker":
		return kind
	default:
		return "message"
	}
}

func deriveDecisions(records []store.EvidenceRecord, now string) []store.EvidenceRecord {
	keywords := []string{"决定", "选择", "方案", "采用", "不采用", "不要", "必须", "decision", "decided", "chosen approach", "we will", "we should"}
	var out []store.EvidenceRecord
	for _, rec := range records {
		if rec.Kind == "decision" {
			continue
		}
		textLower := rec.Text
		matched := ""
		for _, kw := range keywords {
			if containsFold(textLower, kw) {
				matched = kw
				break
			}
		}
		if matched == "" {
			continue
		}
		derived := rec
		derived.ID = store.StableID("derived-decision", rec.ID, matched)
		derived.Kind = "decision"
		derived.IngestedAt = now
		derived.Source.Type = "derived"
		if derived.Meta == nil {
			derived.Meta = map[string]any{}
		}
		derived.Meta["derived_from"] = rec.ID
		derived.Meta["matched_keyword"] = matched
		out = append(out, derived)
	}
	return out
}

func containsFold(s, substr string) bool {
	return substr == "" || strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
