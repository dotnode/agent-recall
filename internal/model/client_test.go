package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-recall/internal/config"
)

func TestOpenAICompatibleClientChatCompletion(t *testing.T) {
	var seen struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"answer text"}}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1")
	res, err := client.ChatCompletion(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hello"}}, Temperature: 0.2, MaxTokens: 10})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if res.Content != "answer text" {
		t.Fatalf("content = %q", res.Content)
	}
	if seen.Model != "test-model" {
		t.Fatalf("model = %q", seen.Model)
	}
	if len(seen.Messages) != 1 || seen.Messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", seen.Messages)
	}
}

func TestOpenAICompatibleClientNon2XXRedactsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad test-key", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("error = %v, want redacted marker", err)
	}
}

func TestOpenAICompatibleClientEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("error = %v, want no choices", err)
	}
}

func TestOpenAICompatibleClientMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1")
	_, err := client.ChatCompletion(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
	if err == nil || !strings.Contains(err.Error(), "decode model response") {
		t.Fatalf("error = %v, want decode error", err)
	}
}

func TestChatCompletionsEndpoint(t *testing.T) {
	cases := map[string]string{
		"https://models.example/v1":                  "https://models.example/v1/chat/completions",
		"https://models.example/v1/":                 "https://models.example/v1/chat/completions",
		"https://models.example/v1/chat/completions": "https://models.example/v1/chat/completions",
	}
	for raw, want := range cases {
		got, err := chatCompletionsEndpoint(raw)
		if err != nil {
			t.Fatalf("chatCompletionsEndpoint(%q) error = %v", raw, err)
		}
		if got != want {
			t.Fatalf("chatCompletionsEndpoint(%q) = %q, want %q", raw, got, want)
		}
	}
}

func newTestClient(t *testing.T, baseURL string) *OpenAICompatibleClient {
	t.Helper()
	client, err := NewOpenAICompatibleClient(config.ModelConfig{
		Provider: config.ModelProviderOpenAICompatible,
		BaseURL:  baseURL,
		APIKey:   "test-key",
		Model:    "test-model",
		Timeout:  time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("NewOpenAICompatibleClient() error = %v", err)
	}
	return client
}
