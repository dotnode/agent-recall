package transcript

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"time"
)

type Line struct {
	Number    int64
	ByteStart int64
	ByteEnd   int64
	Raw       []byte
}

type ReadResult struct {
	Lines   []Line
	Offset  int64
	Line    int64
	Size    int64
	ModTime time.Time
}

func ReadNew(path string, offset, line int64) (ReadResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ReadResult{}, err
	}
	if info.Size() < offset {
		offset = 0
		line = 0
	}
	f, err := os.Open(path)
	if err != nil {
		return ReadResult{}, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return ReadResult{}, err
	}

	res := ReadResult{Offset: offset, Line: line, Size: info.Size(), ModTime: info.ModTime()}
	r := bufio.NewReader(f)
	for {
		start := res.Offset
		b, err := r.ReadBytes('\n')
		if len(b) > 0 {
			if err == io.EOF && !bytes.HasSuffix(b, []byte("\n")) {
				break
			}
			trimmed := bytes.TrimRight(b, "\r\n")
			res.Line++
			res.Offset += int64(len(b))
			if len(trimmed) > 0 {
				res.Lines = append(res.Lines, Line{Number: res.Line, ByteStart: start, ByteEnd: res.Offset, Raw: append([]byte(nil), trimmed...)})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
