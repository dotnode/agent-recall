package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-recall/internal/config"
	"agent-recall/internal/diagnostics"
	"agent-recall/internal/store"
)

func TestToolsListModelDisabled(t *testing.T) {
	clearModelEnv(t)

	if hasTool(tools(io.Discard), "search_answer") {
		t.Fatalf("search_answer should not be listed without model config")
	}
	if !hasTool(tools(io.Discard), "status") {
		t.Fatalf("status should always be listed")
	}
}

func TestToolsListModelEnabled(t *testing.T) {
	setModelEnv(t, "https://models.example/v1")

	if !hasTool(tools(io.Discard), "search_answer") {
		t.Fatalf("search_answer should be listed with model config")
	}
}

func TestToolsListModelConfigError(t *testing.T) {
	clearModelEnv(t)
	t.Setenv(config.EnvModelBaseURL, "https://models.example/v1")

	if hasTool(tools(io.Discard), "search_answer") {
		t.Fatalf("search_answer should not be listed with incomplete model config")
	}
}

func TestCallStatusToolReportsComponents(t *testing.T) {
	clearModelEnv(t)
	result, err := callTool(t.TempDir(), []byte(`{"name":"status","arguments":{}}`))
	if err != nil {
		t.Fatalf("callTool(status) error = %v", err)
	}
	status := decodeTextResult[diagnostics.Status](t, result)
	if status.Components["mcp"].State != diagnostics.StateOK {
		t.Fatalf("mcp component = %+v", status.Components["mcp"])
	}
	if status.Components["model"].State != diagnostics.StateDisabled {
		t.Fatalf("model component = %+v", status.Components["model"])
	}
}

func TestCallSearchAnswer(t *testing.T) {
	var called atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Use OAuth based on the cited evidence."}}]}`))
	}))
	defer server.Close()
	setModelEnv(t, server.URL+"/v1")

	dir := t.TempDir()
	if err := store.AppendRecords(dir, []store.EvidenceRecord{{
		Version:    1,
		ID:         "ev-1",
		Agent:      "claude-code",
		Timestamp:  "2026-05-16T00:00:00Z",
		IngestedAt: time.Now().UTC().Format(time.RFC3339),
		Role:       "user",
		Kind:       "message",
		Text:       "We decided that auth should use OAuth.",
		Source:     store.EvidenceSource{Type: "transcript", TranscriptPath: "session.jsonl", Line: 7},
	}}); err != nil {
		t.Fatalf("AppendRecords() error = %v", err)
	}

	result, err := callTool(dir, []byte(`{"name":"search_answer","arguments":{"query":"auth","limit":5}}`))
	if err != nil {
		t.Fatalf("callTool() error = %v", err)
	}
	answer := decodeTextResult[searchAnswerResult](t, result)
	if answer.Answer != "Use OAuth based on the cited evidence." {
		t.Fatalf("answer = %q", answer.Answer)
	}
	if answer.Model.Name != "test-model" || answer.Model.Provider != config.ModelProviderOpenAICompatible {
		t.Fatalf("model = %+v", answer.Model)
	}
	if len(answer.Items) != 1 || answer.Items[0].ID != "ev-1" {
		t.Fatalf("items = %+v", answer.Items)
	}
	if called.Load() != 1 {
		t.Fatalf("model calls = %d, want 1", called.Load())
	}
}

func TestCallSearchAnswerNoEvidenceDoesNotCallModel(t *testing.T) {
	var called atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"unexpected"}}]}`))
	}))
	defer server.Close()
	setModelEnv(t, server.URL+"/v1")

	result, err := callTool(t.TempDir(), []byte(`{"name":"search_answer","arguments":{"query":"missing"}}`))
	if err != nil {
		t.Fatalf("callTool() error = %v", err)
	}
	answer := decodeTextResult[searchAnswerResult](t, result)
	if len(answer.Items) != 0 {
		t.Fatalf("items = %+v, want none", answer.Items)
	}
	if !strings.Contains(answer.Answer, "没有找到") {
		t.Fatalf("answer = %q", answer.Answer)
	}
	if called.Load() != 0 {
		t.Fatalf("model calls = %d, want 0", called.Load())
	}
}

func TestRecallToolRemainsEvidenceOnly(t *testing.T) {
	clearModelEnv(t)
	dir := t.TempDir()
	if err := store.AppendRecords(dir, []store.EvidenceRecord{{
		Version:    1,
		ID:         "ev-1",
		Agent:      "claude-code",
		IngestedAt: time.Now().UTC().Format(time.RFC3339),
		Role:       "user",
		Kind:       "message",
		Text:       "auth context",
		Source:     store.EvidenceSource{Type: "transcript", TranscriptPath: "session.jsonl", Line: 1},
	}}); err != nil {
		t.Fatalf("AppendRecords() error = %v", err)
	}

	result, err := callTool(dir, []byte(`{"name":"recall","arguments":{"query":"auth"}}`))
	if err != nil {
		t.Fatalf("callTool() error = %v", err)
	}
	var recalled struct {
		Notice string `json:"notice"`
		Items  []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	decodeInto(t, result, &recalled)
	if recalled.Notice != store.Notice {
		t.Fatalf("notice = %q", recalled.Notice)
	}
	if len(recalled.Items) != 1 || recalled.Items[0].ID != "ev-1" {
		t.Fatalf("items = %+v", recalled.Items)
	}
}

func hasTool(list []map[string]any, name string) bool {
	for _, item := range list {
		if item["name"] == name {
			return true
		}
	}
	return false
}

func decodeTextResult[T any](t *testing.T, result any) T {
	t.Helper()
	var out T
	decodeInto(t, result, &out)
	return out
}

func decodeInto(t *testing.T, result any, out any) {
	t.Helper()
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", result)
	}
	content, ok := m["content"].([]map[string]string)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v", m["content"])
	}
	if err := json.Unmarshal([]byte(content[0]["text"]), out); err != nil {
		t.Fatalf("unmarshal text result: %v", err)
	}
}

func setModelEnv(t *testing.T, baseURL string) {
	t.Helper()
	clearModelEnv(t)
	t.Setenv(config.EnvModelProvider, config.ModelProviderOpenAICompatible)
	t.Setenv(config.EnvModelBaseURL, baseURL)
	t.Setenv(config.EnvModelAPIKey, "test-key")
	t.Setenv(config.EnvModelName, "test-model")
}

func clearModelEnv(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvModelProvider, "")
	t.Setenv(config.EnvModelBaseURL, "")
	t.Setenv(config.EnvModelAPIKey, "")
	t.Setenv(config.EnvModelName, "")
	t.Setenv(config.EnvModelTimeout, "")
}
