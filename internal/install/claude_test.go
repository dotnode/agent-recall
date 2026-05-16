package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCodeInstallsAndIsIdempotent(t *testing.T) {
	cwd := chdirTemp(t)
	storeDir := filepath.Join(cwd, "store")
	var out bytes.Buffer
	if err := ClaudeCode(Options{StoreDir: storeDir}, &out); err != nil {
		t.Fatalf("ClaudeCode() error = %v", err)
	}
	if !strings.Contains(out.String(), "installed") {
		t.Fatalf("output = %q", out.String())
	}
	settings, err := readJSON(filepath.Join(cwd, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if !settingsContainCommand(settings, "Stop", "--strict") || !settingsContainCommand(settings, "PreCompact", "--strict") {
		t.Fatalf("hooks should be strict: %+v", settings)
	}

	out.Reset()
	if err := ClaudeCode(Options{StoreDir: storeDir}, &out); err != nil {
		t.Fatalf("ClaudeCode() second error = %v", err)
	}
	settings, err = readJSON(filepath.Join(cwd, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatalf("read settings second: %v", err)
	}
	if hookCount(settings, "Stop") != 1 || hookCount(settings, "PreCompact") != 1 {
		t.Fatalf("settings after second install = %+v", settings)
	}

	mcp, err := readJSON(filepath.Join(cwd, ".mcp.json"))
	if err != nil {
		t.Fatalf("read mcp: %v", err)
	}
	servers := mcp["mcpServers"].(map[string]any)
	server := servers["agent-recall"].(map[string]any)
	env := server["env"].(map[string]any)
	if env["AGENT_RECALL_HOME"] != storeDir {
		t.Fatalf("mcp env = %+v", env)
	}
}

func TestClaudeCodeDryRunDoesNotWriteFiles(t *testing.T) {
	cwd := chdirTemp(t)
	var out bytes.Buffer
	if err := ClaudeCode(Options{DryRun: true}, &out); err != nil {
		t.Fatalf("ClaudeCode() error = %v", err)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Fatalf("output = %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "settings.local.json")); !os.IsNotExist(err) {
		t.Fatalf("settings stat error = %v, want not exist", err)
	}
}

func TestClaudeCodeForceControlsTemplates(t *testing.T) {
	cwd := chdirTemp(t)
	path := filepath.Join(cwd, ".claude", "commands", "recall-session.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("custom"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := ClaudeCode(Options{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("ClaudeCode() error = %v", err)
	}
	if got := readText(t, path); got != "custom" {
		t.Fatalf("template = %q, want custom", got)
	}
	if err := ClaudeCode(Options{Force: true}, &bytes.Buffer{}); err != nil {
		t.Fatalf("ClaudeCode() force error = %v", err)
	}
	if got := readText(t, path); !strings.Contains(got, "agent-recall MCP server") {
		t.Fatalf("template after force = %q", got)
	}
}

func TestEnsureHookUpgradesExistingAgentRecallHookToStrict(t *testing.T) {
	settings := map[string]any{"hooks": map[string]any{"Stop": []any{map[string]any{
		"matcher": "",
		"hooks":   []any{map[string]any{"type": "command", "command": "agent-recall hook-sync"}},
	}}}}
	changed, err := ensureHook(settings, "Stop", "agent-recall hook-sync --strict")
	if err != nil {
		t.Fatalf("ensureHook() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected existing hook to be upgraded")
	}
	if hookCount(settings, "Stop") != 1 || !settingsContainCommand(settings, "Stop", "--strict") {
		t.Fatalf("settings = %+v", settings)
	}
}

func TestEnsureHookDoesNotRewriteNonAgentRecallHook(t *testing.T) {
	settings := map[string]any{"hooks": map[string]any{"Stop": []any{map[string]any{
		"matcher": "",
		"hooks":   []any{map[string]any{"type": "command", "command": "echo not-agent-recall hook-sync"}},
	}}}}
	changed, err := ensureHook(settings, "Stop", "agent-recall hook-sync")
	if err != nil {
		t.Fatalf("ensureHook() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected new agent-recall hook")
	}
	if hookCount(settings, "Stop") != 2 {
		t.Fatalf("settings = %+v", settings)
	}
}

func TestAgentRecallHookCommandRecognizesQuotedPath(t *testing.T) {
	if !isAgentRecallHookCommand(`"/tmp/path with space/agent-recall" hook-sync --store-dir "/tmp/store"`, "hook-sync") {
		t.Fatalf("expected quoted path command to match")
	}
	if !isAgentRecallHookCommand(`C:\Tools\agent-recall.exe hook-sync`, "hook-sync") {
		t.Fatalf("expected Windows path command to match")
	}
	if isAgentRecallHookCommand("echo agent-recall hook-sync", "hook-sync") {
		t.Fatalf("echo command should not match")
	}
	if isAgentRecallHookCommand("agent-recall hook-flush", "hook-sync") {
		t.Fatalf("wrong subcommand should not match")
	}
}

func hookCount(settings map[string]any, event string) int {
	hooks := settings["hooks"].(map[string]any)
	arr := hooks[event].([]any)
	count := 0
	for _, item := range arr {
		m := item.(map[string]any)
		count += len(m["hooks"].([]any))
	}
	return count
}

func settingsContainCommand(settings map[string]any, event, substr string) bool {
	hooks := settings["hooks"].(map[string]any)
	arr := hooks[event].([]any)
	for _, item := range arr {
		m := item.(map[string]any)
		for _, h := range m["hooks"].([]any) {
			hm := h.(map[string]any)
			if strings.Contains(hm["command"].(string), substr) {
				return true
			}
		}
	}
	return false
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	cwd := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
	return cwd
}

func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(b)
}
