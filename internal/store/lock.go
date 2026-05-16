package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Lock struct {
	path string
}

func AcquireLock(dir string) (*Lock, error) {
	if err := Ensure(dir); err != nil {
		return nil, err
	}
	path := LockPath(dir)
	for i := 0; i < 20; i++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "pid=%d\ntime=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
			_ = f.Close()
			return &Lock{path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > time.Minute {
			_ = os.Remove(path)
			continue
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("store is locked: %s", filepath.Clean(path))
}

func (l *Lock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	return os.Remove(l.path)
}
