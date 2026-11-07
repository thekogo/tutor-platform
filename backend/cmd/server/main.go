// Command server runs the tutor platform backend: REST API under /api, MCP at
// /mcp, and health probes. Subcommands:
//
//	server [serve]                  run the HTTP server (migrations run on start)
//	server seed -file graph.yaml    import a course graph (idempotent upsert by slug)
//	server rebuild-mastery          recompute every Learner's Mastery from Answers
//	server migrate                  apply database migrations and exit
//	server healthcheck              GET /readyz on the local server; exit 0/1 (container healthcheck)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/auth"
	"github.com/thekogo/tutor-platform/backend/internal/httpapi"
	"github.com/thekogo/tutor-platform/backend/internal/mcpserver"
	"github.com/thekogo/tutor-platform/backend/internal/seed"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	cmd := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "serve":
		err = serve(ctx, cfg, log)
	case "seed":
		err = runSeed(ctx, cfg, args, log)
	case "rebuild-mastery":
		err = rebuildMastery(ctx, cfg, log)
	case "migrate":
		err = migrate(ctx, cfg)
	case "healthcheck":
		err = healthcheck(ctx, cfg)
	default:
		err = fmt.Errorf("unknown command %q (serve, seed, rebuild-mastery, migrate, healthcheck)", cmd)
	}
	if err != nil {
		log.Error("exit", "command", cmd, "error", err)
		stop()
		os.Exit(1) //nolint:gocritic // deferred stop already called explicitly
	}
}

func openDB(ctx context.Context, cfg Config) (*app.Service, func(), error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, errors.New("DATABASE_URL is required")
	}
	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, nil, err
	}
	return app.New(pool, slog.Default()), pool.Close, nil
}

func serve(ctx context.Context, cfg Config, log *slog.Logger) error {
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}
	svc := app.New(pool, log)

	var idp auth.IdentityProvider
	if g := auth.NewGoogle(auth.GoogleConfig{
		ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret, RedirectURL: cfg.GoogleRedirectURL,
	}); g != nil {
		idp = g
	} else {
		log.Warn("Google login disabled: GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REDIRECT_URL not set")
	}
	if cfg.DevLogin {
		log.Warn("AUTH_DEV_LOGIN=true: dev login is enabled — never use in production")
	}
	if cfg.MCPToken == "" {
		log.Warn("MCP_TOKEN not set: /mcp is disabled")
	}
	authSvc := auth.New(pool, auth.Config{AppBaseURL: cfg.AppBaseURL, CookieSecure: cfg.CookieSecure, DevLogin: cfg.DevLogin}, idp)

	mux := http.NewServeMux()
	mux.Handle("/api/", httpapi.New(svc, authSvc, httpapi.Config{AppBaseURL: cfg.AppBaseURL}, log))
	mux.Handle("/mcp", http.MaxBytesHandler(mcpserver.Handler(svc, cfg.MCPToken, log), 8<<20))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := svc.Ping(pctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.Common(mux, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	// Periodically purge expired sessions.
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := authSvc.PurgeExpiredSessions(ctx); err != nil {
					log.Warn("purge sessions", "error", err)
				} else if n > 0 {
					log.Info("purged expired sessions", "count", n)
				}
			}
		}
	}()

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	return srv.Shutdown(sctx)
}

func runSeed(ctx context.Context, cfg Config, args []string, log *slog.Logger) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	file := fs.String("file", "", "path to content/<course>/graph.yaml")
	archive := fs.Bool("archive-missing", false, "archive live Concepts absent from the file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("seed: -file is required")
	}
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	_, err := seed.Run(ctx, cfg.DatabaseURL, *file, *archive, log)
	return err
}

func rebuildMastery(ctx context.Context, cfg Config, log *slog.Logger) error {
	svc, closeFn, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeFn()
	n, err := svc.RebuildAllMastery(ctx)
	if err != nil {
		return err
	}
	log.Info("rebuilt mastery", "learner_courses", n)
	return nil
}

func migrate(ctx context.Context, cfg Config) error {
	_, closeFn, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	closeFn()
	return nil
}

func healthcheck(ctx context.Context, cfg Config) error {
	_, port, err := net.SplitHostPort(cfg.HTTPAddr)
	if err != nil || port == "" {
		port = "8080"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/readyz", http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("readyz returned %d", resp.StatusCode)
	}
	return nil
}
