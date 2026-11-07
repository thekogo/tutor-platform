package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

type ctxKey int

const (
	keyLearner ctxKey = iota
	keyWriter
	keyRequest
	keyRequestID
)

// RequestID returns the request ID stored in ctx.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(keyRequestID).(string)
	return id
}

// writeError writes the consistent Error JSON body.
func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Error{Code: code, Message: msg})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Common wraps the whole server (REST, MCP, probes) with request IDs, a JSON
// access log, and panic recovery.
func Common(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), keyRequestID, id)
		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler { //nolint:errorlint // sentinel compared by identity, as net/http does
					panic(p)
				}
				log.ErrorContext(ctx, "panic", "request_id", id, "panic", p, "stack", string(debug.Stack()))
				if rec.status == 0 {
					writeError(rec, http.StatusInternalServerError, "internal", "internal server error")
				}
			}
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				return
			}
			log.InfoContext(ctx, "http",
				"request_id", id, "method", r.Method, "path", r.URL.Path, "status", rec.status,
				"bytes", rec.bytes, "duration_ms", time.Since(start).Milliseconds(), "remote", r.RemoteAddr)
		}()
		next.ServeHTTP(rec, r.WithContext(ctx))
	})
}

// apiMiddleware applies REST-only concerns: body size limit, timeout, Origin
// check on unsafe methods, session loading, and exposing the ResponseWriter
// to handlers (for cookies/redirects).
func (s *Server) apiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
		defer cancel()
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if !s.originAllowed(r) {
				writeError(w, http.StatusForbidden, "bad_origin", "cross-origin request rejected")
				return
			}
		}
		if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
			if l, err := s.auth.Authenticate(ctx, c.Value); err == nil {
				ctx = context.WithValue(ctx, keyLearner, l)
			}
		}
		ctx = context.WithValue(ctx, keyWriter, w)
		ctx = context.WithValue(ctx, keyRequest, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	return strings.EqualFold(strings.TrimRight(origin, "/"), s.allowedOrigin)
}

func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	return u.Scheme + "://" + u.Host
}

// requireAuth enforces the spec's `session` security requirement: operations
// that declare it (the generated wrapper sets SessionScopes) need a Learner.
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(SessionScopes) != nil {
			if _, ok := r.Context().Value(keyLearner).(store.Learner); !ok {
				writeError(w, http.StatusUnauthorized, "unauthenticated", "login required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
