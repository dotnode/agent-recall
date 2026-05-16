package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadNewReadsFromOffsetAndSkipsEmptyLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := "{\"n\":1}\n\n{\"n\":2}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	first, err := ReadNew(path, 0, 0)
	if err != nil {
		t.Fatalf("ReadNew() error = %v", err)
	}
	if len(first.Lines) != 2 || first.Line != 3 || first.Offset != int64(len(content)) {
		t.Fatalf("first = %+v", first)
	}

	more := "{\"n\":3}\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := f.WriteString(more); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	second, err := ReadNew(path, first.Offset, first.Line)
	if err != nil {
		t.Fatalf("ReadNew() second error = %v", err)
	}
	if len(second.Lines) != 1 || string(second.Lines[0].Raw) != "{\"n\":3}" || second.Line != 4 {
		t.Fatalf("second = %+v", second)
	}
}

func TestReadNewResetsAfterTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("{\"n\":1}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	res, err := ReadNew(path, 999, 99)
	if err != nil {
		t.Fatalf("ReadNew() error = %v", err)
	}
	if len(res.Lines) != 1 || res.Lines[0].Number != 1 || res.Offset != 8 {
		t.Fatalf("res = %+v", res)
	}
}

func TestReadNewIgnoresIncompleteEOFLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("{\"n\":1}\n{\"partial\":true"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	res, err := ReadNew(path, 0, 0)
	if err != nil {
		t.Fatalf("ReadNew() error = %v", err)
	}
	if len(res.Lines) != 1 || res.Line != 1 || res.Offset != 8 {
		t.Fatalf("res = %+v", res)
	}
}
