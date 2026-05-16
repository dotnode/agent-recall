package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const EnvHome = "AGENT_RECALL_HOME"

type Options struct {
	StoreDir string
}

func StoreDir(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if env := os.Getenv(EnvHome); env != "" {
		return filepath.Abs(env)
	}
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "agent-recall"), nil
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, "Library", "Application Support", "agent-recall"), nil
		}
	default:
		if state := os.Getenv("XDG_STATE_HOME"); state != "" {
			return filepath.Join(state, "agent-recall"), nil
		}
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, ".local", "state", "agent-recall"), nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".agent-recall"), nil
}
