package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is read from the environment (12-factor). Names are part of the
// deployment contract (docker-compose.yml, .env.example).
type Config struct {
	DatabaseURL        string
	HTTPAddr           string
	AppBaseURL         string
	AllowedOrigins     []string // Origins allowed on unsafe methods; ["*"] allows any
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	CookieSecure       bool
	DevLogin           bool
	MCPToken           string
	LogLevel           slog.Level

	// Tutor Chat: any OpenAI-compatible chat completions gateway.
	AIBaseURL     string
	AIAPIKey      string
	AIModel       string
	AITimeout     time.Duration
	AIMaxTokens   int
	AIExtraBody   map[string]any
	AIHelpPerHour int
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) (bool, error) {
	v := env(key, "")
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func envInt(key string, def int) (int, error) {
	v := env(key, "")
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s: want a positive integer, got %q", key, v)
	}
	return n, nil
}

func loadConfig() (Config, error) {
	c := Config{
		DatabaseURL:        env("DATABASE_URL", ""),
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		AppBaseURL:         strings.TrimRight(env("APP_BASE_URL", "http://localhost:3000"), "/"),
		GoogleClientID:     env("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: env("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  env("GOOGLE_REDIRECT_URL", ""),
		MCPToken:           env("MCP_TOKEN", ""),
		AIBaseURL:          env("AI_BASE_URL", ""),
		AIAPIKey:           env("AI_API_KEY", ""),
		AIModel:            env("AI_MODEL", ""),
	}
	for _, o := range strings.Split(env("ALLOWED_ORIGINS", c.AppBaseURL), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}
	var err error
	if c.CookieSecure, err = envBool("SESSION_COOKIE_SECURE", true); err != nil {
		return c, err
	}
	if c.DevLogin, err = envBool("AUTH_DEV_LOGIN", false); err != nil {
		return c, err
	}
	if c.AITimeout, err = time.ParseDuration(env("AI_TIMEOUT", "25s")); err != nil || c.AITimeout <= 0 {
		return c, fmt.Errorf("AI_TIMEOUT: want a positive duration such as 25s, got %q", os.Getenv("AI_TIMEOUT"))
	}
	if c.AIMaxTokens, err = envInt("AI_MAX_TOKENS", 800); err != nil {
		return c, err
	}
	if c.AIHelpPerHour, err = envInt("AI_HELP_PER_HOUR", 30); err != nil {
		return c, err
	}
	if raw := env("AI_EXTRA_BODY", ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &c.AIExtraBody); err != nil {
			return c, fmt.Errorf("AI_EXTRA_BODY: must be a JSON object: %w", err)
		}
	}
	switch strings.ToLower(env("LOG_LEVEL", "info")) {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "info":
		c.LogLevel = slog.LevelInfo
	case "warn", "warning":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		return c, fmt.Errorf("LOG_LEVEL: unknown level %q", os.Getenv("LOG_LEVEL"))
	}
	return c, nil
}
