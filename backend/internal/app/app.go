// Package app holds the use-case services of the tutor platform. Both
// transports (REST in httpapi, MCP in mcpserver) call these services; neither
// contains business logic. app owns transaction boundaries and authorization,
// maps store rows into domain types, and calls the pure domain engine.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thekogo/tutor-platform/backend/internal/domain"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Session sizing and placement parameters.
const (
	QuizSize           = 10
	DiagnosticSize     = 4
	PlacementMaxProbes = 18
)

// Actor is the authenticated Learner performing a REST use case.
type Actor struct {
	LearnerID uuid.UUID
	IsOwner   bool
}

// Service is the application layer.
type Service struct {
	pool       *pgxpool.Pool
	q          *store.Queries
	log        *slog.Logger
	masteryCfg domain.MasteryConfig
	// placementCfg estimates Mastery from the few probe Answers of a single
	// Placement Test; it only steers probe selection (see placement.go).
	placementCfg domain.MasteryConfig
	now          func() time.Time

	mu     sync.RWMutex
	graphs map[uuid.UUID]*CourseGraph
}

// New builds a Service.
func New(pool *pgxpool.Pool, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	pc := domain.DefaultMasteryConfig()
	pc.MinEvidence = 0.5
	return &Service{
		pool:         pool,
		q:            store.New(pool),
		log:          log,
		masteryCfg:   domain.DefaultMasteryConfig(),
		placementCfg: pc,
		now:          time.Now,
		graphs:       map[uuid.UUID]*CourseGraph{},
	}
}

// SetClock overrides the clock (tests).
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// Ping checks database connectivity (readiness).
func (s *Service) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Service) tx(ctx context.Context, fn func(q *store.Queries) error) error {
	return store.InTx(ctx, s.pool, fn)
}

// ── Errors ──────────────────────────────────────────────────────────────

// Kind classifies application errors; transports map it to status codes.
type Kind int

const (
	KindInvalid      Kind = iota + 1 // 422
	KindBadRequest                   // 400
	KindUnauthorized                 // 401
	KindForbidden                    // 403
	KindNotFound                     // 404
	KindConflict                     // 409
	KindGone                         // 410
	KindUnavailable                  // 503
)

// Error is an expected, client-facing error.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Details []string
}

func (e *Error) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Details)
	}
	return e.Code + ": " + e.Message
}

// E builds an *Error.
func E(kind Kind, code, format string, args ...any) *Error {
	return &Error{Kind: kind, Code: code, Message: fmt.Sprintf(format, args...)}
}

// AsError extracts an *Error from err.
func AsError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

func notFound(what string) *Error { return E(KindNotFound, "not_found", "%s not found", what) }

func forbidden(msg string) *Error { return E(KindForbidden, "forbidden", "%s", msg) }

// mapNotFound converts "no rows" into a not_found Error.
func mapNotFound(err error, what string) error {
	if store.IsNotFound(err) {
		return notFound(what)
	}
	return err
}

func requireOwner(a Actor) error {
	if !a.IsOwner {
		return forbidden("only the Owner may do this")
	}
	return nil
}

// ── ID helpers ──────────────────────────────────────────────────────────

func cid(id uuid.UUID) domain.ConceptID { return domain.ConceptID(id.String()) }

func tid(id uuid.UUID) domain.TopicID { return domain.TopicID(id.String()) }

func ucid(id domain.ConceptID) uuid.UUID { return uuid.MustParse(string(id)) }

func ptr[T any](v T) *T { return &v }
