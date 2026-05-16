package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"agent-recall/internal/config"
)

type Client interface {
	ChatCompletion(context.Context, ChatRequest) (ChatResponse, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Content string
}

type OpenAICompatibleClient struct {
	cfg        config.ModelConfig
	httpClient *http.Client
	endpoint   string
}

func NewOpenAICompatibleClient(cfg config.ModelConfig, httpClient *http.Client) (*OpenAICompatibleClient, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	endpoint, err := chatCompletionsEndpoint(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	return &OpenAICompatibleClient{cfg: cfg, httpClient: httpClient, endpoint: endpoint}, nil
}

func (c *OpenAICompatibleClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature,omitempty"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
	}{
		Model:       c.cfg.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return ChatResponse{}, fmt.Errorf("model request failed with status %d: %s", res.StatusCode, sanitizeErrorBody(string(b), c.cfg.APIKey))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("decode model response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("model response had no choices")
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return ChatResponse{}, fmt.Errorf("model response content was empty")
	}
	return ChatResponse{Content: content}, nil
}

func chatCompletionsEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid model base URL")
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid model base URL")
	}
	path := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(path, "/chat/completions") {
		u.Path = path
	} else {
		u.Path = path + "/chat/completions"
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func sanitizeErrorBody(body, apiKey string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return http.StatusText(http.StatusInternalServerError)
	}
	if apiKey != "" {
		body = strings.ReplaceAll(body, apiKey, "[redacted]")
	}
	return body
}
