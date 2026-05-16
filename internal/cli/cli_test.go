package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"agent-recall/internal/config"
	"agent-recall/internal/diagnostics"
	"agent-recall/internal/version"
)

func TestRunHelpAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Run(nil, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help stdout = %q", stdout.String())
	}

	stdout.Reset()
	if err := Run([]string{"version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("Run(version) error = %v", err)
	}
	if strings.TrimSpace(stdout.String()) != version.Version {
		t.Fatalf("version stdout = %q", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := Run([]string{"unknown"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunRecallRequiresQuery(t *testing.T) {
	err := Run([]string{"recall", "--store-dir", t.TempDir()}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "requires a query") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunStatusJSONReportsComponents(t *testing.T) {
	clearModelEnv(t)
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"status", "--store-dir", t.TempDir(), "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("Run(status) error = %v", err)
	}
	var status diagnostics.Status
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if status.Components["model"].State != diagnostics.StateDisabled {
		t.Fatalf("model component = %+v", status.Components["model"])
	}
	if status.Components["mcp"].State != diagnostics.StateNotChecked {
		t.Fatalf("mcp component = %+v", status.Components["mcp"])
	}
}

func TestRunStatusReportsPartialModelConfig(t *testing.T) {
	clearModelEnv(t)
	t.Setenv(config.EnvModelBaseURL, "https://models.example/v1")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"status", "--store-dir", t.TempDir(), "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("Run(status) error = %v", err)
	}
	var status diagnostics.Status
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	model := status.Components["model"]
	if model.State != diagnostics.StateError || !strings.Contains(model.Message, config.EnvModelProvider) {
		t.Fatalf("model component = %+v", model)
	}
}

func TestRunHookSoftFailAndStrict(t *testing.T) {
	input := `{"hook_event_name":"Stop","transcript_path":"/missing/transcript.jsonl"}`
	var stderr bytes.Buffer
	if err := Run([]string{"hook-sync", "--store-dir", t.TempDir()}, strings.NewReader(input), &bytes.Buffer{}, &stderr); err != nil {
		t.Fatalf("soft hook error = %v", err)
	}
	if !strings.Contains(stderr.String(), "agent-recall hook error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stderr.Reset()
	err := Run([]string{"hook-sync", "--store-dir", t.TempDir(), "--strict"}, strings.NewReader(input), &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatalf("strict hook error = nil")
	}
	if !strings.Contains(stderr.String(), "agent-recall hook error:") {
		t.Fatalf("strict stderr = %q", stderr.String())
	}
}

func clearModelEnv(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvModelProvider, "")
	t.Setenv(config.EnvModelBaseURL, "")
	t.Setenv(config.EnvModelAPIKey, "")
	t.Setenv(config.EnvModelName, "")
	t.Setenv(config.EnvModelTimeout, "")
}
