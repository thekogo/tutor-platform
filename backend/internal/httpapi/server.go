// Package httpapi implements the REST API (api/openapi.yaml) with an
// oapi-codegen strict server. Handlers are thin: they translate between the
// generated types and app services. Regenerate with `make generate`.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/auth"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

const sessionCookieName = auth.SessionCookie

// Config configures the REST API.
type Config struct {
	AppBaseURL     string   // redirect target after login
	AllowedOrigins []string // Origins allowed on unsafe methods; empty = AppBaseURL's origin; "*" allows any
	MaxBodyBytes   int64
	RequestTimeout time.Duration
}

// Server implements StrictServerInterface.
type Server struct {
	app            *app.Service
	auth           *auth.Service
	cfg            Config
	log            *slog.Logger
	allowedOrigins map[string]bool
	allowAnyOrigin bool
}

var _ StrictServerInterface = (*Server)(nil)

// New builds the REST handler, to be mounted at /api/ (paths include /api).
func New(svc *app.Service, authSvc *auth.Service, cfg Config, log *slog.Logger) http.Handler {
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	s := &Server{app: svc, auth: authSvc, cfg: cfg, log: log, allowedOrigins: map[string]bool{}}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{cfg.AppBaseURL}
	}
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			s.allowAnyOrigin = true
			continue
		}
		s.allowedOrigins[strings.ToLower(originOf(o))] = true
	}
	strict := NewStrictHandlerWithOptions(s, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			var mbe *http.MaxBytesError
			if errors.As(err, &mbe) {
				writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", "malformed request body")
		},
		ResponseErrorHandlerFunc: s.handleError,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
	})
	h := HandlerWithOptions(strict, StdHTTPServerOptions{
		BaseURL:     "/api",
		BaseRouter:  mux,
		Middlewares: []MiddlewareFunc{requireAuth},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		},
	})
	return s.apiMiddleware(h)
}

// statusFor maps app error kinds to HTTP status codes.
func statusFor(k app.Kind) int {
	switch k {
	case app.KindInvalid:
		return http.StatusUnprocessableEntity
	case app.KindBadRequest:
		return http.StatusBadRequest
	case app.KindUnauthorized:
		return http.StatusUnauthorized
	case app.KindForbidden:
		return http.StatusForbidden
	case app.KindNotFound:
		return http.StatusNotFound
	case app.KindConflict:
		return http.StatusConflict
	case app.KindGone:
		return http.StatusGone
	case app.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := app.AsError(err); ok {
		msg := e.Message
		for _, d := range e.Details {
			msg += "; " + d
		}
		writeError(w, statusFor(e.Kind), e.Code, msg)
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusServiceUnavailable, "timeout", "request timed out")
		return
	}
	s.log.ErrorContext(r.Context(), "request failed", "request_id", RequestID(r.Context()), "error", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}

// ── context helpers ─────────────────────────────────────────────────────

func learnerFrom(ctx context.Context) (store.Learner, bool) {
	l, ok := ctx.Value(keyLearner).(store.Learner)
	return l, ok
}

func actorFrom(ctx context.Context) (app.Actor, error) {
	l, ok := learnerFrom(ctx)
	if !ok {
		return app.Actor{}, app.E(app.KindUnauthorized, "unauthenticated", "login required")
	}
	return app.Actor{LearnerID: l.ID, IsOwner: l.IsOwner}, nil
}

func writerFrom(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(keyWriter).(http.ResponseWriter)
	return w
}

func requestFrom(ctx context.Context) *http.Request {
	r, _ := ctx.Value(keyRequest).(*http.Request)
	return r
}

func setCookie(ctx context.Context, c *http.Cookie) {
	if w := writerFrom(ctx); w != nil && c != nil {
		http.SetCookie(w, c)
	}
}

func setHeader(ctx context.Context, k, v string) {
	if w := writerFrom(ctx); w != nil {
		w.Header().Set(k, v)
	}
}
