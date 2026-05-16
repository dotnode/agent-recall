package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	EnvModelProvider = "AGENT_RECALL_MODEL_PROVIDER"
	EnvModelBaseURL  = "AGENT_RECALL_MODEL_BASE_URL"
	EnvModelAPIKey   = "AGENT_RECALL_MODEL_API_KEY"
	EnvModelName     = "AGENT_RECALL_MODEL_NAME"
	EnvModelTimeout  = "AGENT_RECALL_MODEL_TIMEOUT"

	ModelProviderOpenAICompatible = "openai-compatible"
)

const DefaultModelTimeout = 20 * time.Second

type ModelConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func ModelConfigFromEnv() (ModelConfig, error) {
	cfg := ModelConfig{
		Provider: strings.TrimSpace(os.Getenv(EnvModelProvider)),
		BaseURL:  strings.TrimSpace(os.Getenv(EnvModelBaseURL)),
		APIKey:   strings.TrimSpace(os.Getenv(EnvModelAPIKey)),
		Model:    strings.TrimSpace(os.Getenv(EnvModelName)),
		Timeout:  DefaultModelTimeout,
	}
	rawTimeout := strings.TrimSpace(os.Getenv(EnvModelTimeout))
	anyModelEnv := cfg.Provider != "" || cfg.BaseURL != "" || cfg.APIKey != "" || cfg.Model != "" || rawTimeout != ""
	if rawTimeout != "" {
		timeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return ModelConfig{}, fmt.Errorf("invalid %s", EnvModelTimeout)
		}
		if timeout <= 0 {
			return ModelConfig{}, fmt.Errorf("%s must be positive", EnvModelTimeout)
		}
		cfg.Timeout = timeout
	}
	if !anyModelEnv {
		return cfg, nil
	}
	if cfg.Provider == "" {
		return ModelConfig{}, fmt.Errorf("%s is required when any AGENT_RECALL_MODEL_* variable is set", EnvModelProvider)
	}
	if cfg.Provider != ModelProviderOpenAICompatible {
		return ModelConfig{}, fmt.Errorf("unsupported %s %q", EnvModelProvider, cfg.Provider)
	}
	if cfg.BaseURL == "" {
		return ModelConfig{}, fmt.Errorf("%s is required when %s is set", EnvModelBaseURL, EnvModelProvider)
	}
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return ModelConfig{}, fmt.Errorf("%s must be a valid URL", EnvModelBaseURL)
	}
	if cfg.APIKey == "" {
		return ModelConfig{}, fmt.Errorf("%s is required when %s is set", EnvModelAPIKey, EnvModelProvider)
	}
	if cfg.Model == "" {
		return ModelConfig{}, fmt.Errorf("%s is required when %s is set", EnvModelName, EnvModelProvider)
	}
	return cfg, nil
}

func (c ModelConfig) Enabled() bool {
	return c.Provider != ""
}
