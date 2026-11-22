package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDisabledWithoutConfig(t *testing.T) {
	if New(Config{BaseURL: "http://x", APIKey: "k"}) != nil {
		t.Fatal("client without model should be nil")
	}
}

func TestChatSendsOpenAICompatibleRequest(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/chat/completions" {
			t.Errorf("path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"model":"m1","choices":[{"message":{"role":"assistant","content":"  hint  "}}],"usage":{"prompt_tokens":10,"completion_tokens":3}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/v4/", APIKey: "secret", Model: "m1", ExtraBody: map[string]any{"thinking": map[string]any{"type": "disabled"}, "model": "ignored"}})
	r, err := c.Chat(context.Background(), []Message{{Role: RoleSystem, Content: "s"}, {Role: RoleUser, Content: "u"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Content != "hint" || r.Usage.PromptTokens != 10 || r.Usage.CompletionTokens != 3 {
		t.Fatalf("reply %+v", r)
	}
	if got["model"] != "m1" || got["stream"] != false || got["thinking"] == nil {
		t.Fatalf("body %v", got)
	}
	if msgs, _ := got["messages"].([]any); len(msgs) != 2 {
		t.Fatalf("messages %v", got["messages"])
	}
}

func TestChatUpstreamErrors(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"status": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
		},
		"empty": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"choices":[]}`)) },
		"junk":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`not json`)) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			_, err := New(Config{BaseURL: srv.URL, APIKey: "k", Model: "m"}).Chat(context.Background(), nil)
			if !errors.Is(err, ErrUpstream) {
				t.Fatalf("want ErrUpstream, got %v", err)
			}
		})
	}
}
