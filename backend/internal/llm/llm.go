// Package llm is a minimal client for any OpenAI-compatible chat completions
// gateway (Z.AI, OpenRouter, LiteLLM, Cloudflare AI Gateway, OpenAI, …):
// POST {BaseURL}/chat/completions with a bearer key. It knows nothing about
// the tutor domain; prompts are built by the app layer.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"
)

// Roles of chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage is the token accounting reported by the gateway (zero if absent).
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Reply is the model's answer.
type Reply struct {
	Content string
	Model   string
	Usage   Usage
}

// Chatter completes a conversation.
type Chatter interface {
	Chat(ctx context.Context, msgs []Message) (Reply, error)
}

// Config configures an OpenAI-compatible gateway.
type Config struct {
	BaseURL     string         // e.g. https://api.z.ai/api/paas/v4 (no trailing /chat/completions)
	APIKey      string         // sent as Authorization: Bearer
	Model       string         // e.g. glm-4.6
	Timeout     time.Duration  // per call; default 25s
	MaxTokens   int            // default 800
	Temperature float64        // default 0.3
	ExtraBody   map[string]any // merged into every request body (provider-specific knobs)
}

// Client talks to an OpenAI-compatible gateway.
type Client struct {
	cfg  Config
	http *http.Client
}

var _ Chatter = (*Client)(nil)

// New returns a Client, or nil when BaseURL, APIKey or Model is empty
// (the feature is then disabled).
func New(cfg Config) *Client {
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 25 * time.Second
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 800
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.3
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// ErrUpstream wraps gateway failures (non-2xx, malformed or empty replies).
var ErrUpstream = errors.New("llm gateway error")

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// Chat sends one chat completion request.
func (c *Client) Chat(ctx context.Context, msgs []Message) (Reply, error) {
	body := make(map[string]any, len(c.cfg.ExtraBody)+5)
	maps.Copy(body, c.cfg.ExtraBody)
	body["model"] = c.cfg.Model
	body["messages"] = msgs
	body["max_tokens"] = c.cfg.MaxTokens
	body["temperature"] = c.cfg.Temperature
	body["stream"] = false
	raw, err := json.Marshal(body)
	if err != nil {
		return Reply{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Reply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return Reply{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Reply{}, fmt.Errorf("%w: read body: %w", ErrUpstream, err)
	}
	if resp.StatusCode/100 != 2 {
		snippet := string(data)
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		return Reply{}, fmt.Errorf("%w: status %d: %s", ErrUpstream, resp.StatusCode, snippet)
	}
	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return Reply{}, fmt.Errorf("%w: decode: %w", ErrUpstream, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return Reply{}, fmt.Errorf("%w: empty reply", ErrUpstream)
	}
	return Reply{Content: strings.TrimSpace(out.Choices[0].Message.Content), Model: out.Model, Usage: out.Usage}, nil
}
