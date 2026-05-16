package config

import (
	"strings"
	"testing"
	"time"
)

func TestModelConfigFromEnvDisabled(t *testing.T) {
	clearModelEnv(t)

	cfg, err := ModelConfigFromEnv()
	if err != nil {
		t.Fatalf("ModelConfigFromEnv() error = %v", err)
	}
	if cfg.Enabled() {
		t.Fatalf("expected disabled config")
	}
	if cfg.Timeout != DefaultModelTimeout {
		t.Fatalf("timeout = %v, want %v", cfg.Timeout, DefaultModelTimeout)
	}
}

func TestModelConfigFromEnvOpenAICompatible(t *testing.T) {
	clearModelEnv(t)
	t.Setenv(EnvModelProvider, ModelProviderOpenAICompatible)
	t.Setenv(EnvModelBaseURL, "https://models.example/v1")
	t.Setenv(EnvModelAPIKey, "test-key")
	t.Setenv(EnvModelName, "test-model")
	t.Setenv(EnvModelTimeout, "5s")

	cfg, err := ModelConfigFromEnv()
	if err != nil {
		t.Fatalf("ModelConfigFromEnv() error = %v", err)
	}
	if !cfg.Enabled() {
		t.Fatalf("expected enabled config")
	}
	if cfg.Provider != ModelProviderOpenAICompatible || cfg.BaseURL != "https://models.example/v1" || cfg.APIKey != "test-key" || cfg.Model != "test-model" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", cfg.Timeout)
	}
}

func TestModelConfigFromEnvUnsupportedProvider(t *testing.T) {
	clearModelEnv(t)
	t.Setenv(EnvModelProvider, "anthropic")

	_, err := ModelConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v, want unsupported provider", err)
	}
}

func TestModelConfigFromEnvRequiresFields(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		apiKey  string
		model   string
		want    string
	}{
		{name: "base url", apiKey: "test-key", model: "test-model", want: EnvModelBaseURL},
		{name: "api key", baseURL: "https://models.example/v1", model: "test-model", want: EnvModelAPIKey},
		{name: "model", baseURL: "https://models.example/v1", apiKey: "test-key", want: EnvModelName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearModelEnv(t)
			t.Setenv(EnvModelProvider, ModelProviderOpenAICompatible)
			if tc.baseURL != "" {
				t.Setenv(EnvModelBaseURL, tc.baseURL)
			}
			if tc.apiKey != "" {
				t.Setenv(EnvModelAPIKey, tc.apiKey)
			}
			if tc.model != "" {
				t.Setenv(EnvModelName, tc.model)
			}
			_, err := ModelConfigFromEnv()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestModelConfigFromEnvInvalidTimeout(t *testing.T) {
	clearModelEnv(t)
	t.Setenv(EnvModelTimeout, "soon")

	_, err := ModelConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), EnvModelTimeout) {
		t.Fatalf("error = %v, want timeout error", err)
	}
}

func clearModelEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvModelProvider, "")
	t.Setenv(EnvModelBaseURL, "")
	t.Setenv(EnvModelAPIKey, "")
	t.Setenv(EnvModelName, "")
	t.Setenv(EnvModelTimeout, "")
}
