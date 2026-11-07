package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config is read from the environment (12-factor). Names are part of the
// deployment contract (docker-compose.yml, .env.example).
type Config struct {
	DatabaseURL        string
	HTTPAddr           string
	AppBaseURL         string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	CookieSecure       bool
	DevLogin           bool
	MCPToken           string
	LogLevel           slog.Level
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

func loadConfig() (Config, error) {
	c := Config{
		DatabaseURL:        env("DATABASE_URL", ""),
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		AppBaseURL:         strings.TrimRight(env("APP_BASE_URL", "http://localhost:3000"), "/"),
		GoogleClientID:     env("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: env("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  env("GOOGLE_REDIRECT_URL", ""),
		MCPToken:           env("MCP_TOKEN", ""),
	}
	var err error
	if c.CookieSecure, err = envBool("SESSION_COOKIE_SECURE", true); err != nil {
		return c, err
	}
	if c.DevLogin, err = envBool("AUTH_DEV_LOGIN", false); err != nil {
		return c, err
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
