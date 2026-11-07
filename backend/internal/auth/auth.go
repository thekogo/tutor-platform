// Package auth implements login (Google OIDC, invite-only; dev login for local
// use), opaque session tokens stored as SHA-256 hashes, and invites.
//
// The first-ever Learner to log in becomes the Owner. Everyone else needs a
// valid, unused, unexpired invite token.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Cookie names and lifetimes.
const (
	SessionCookie = "tp_session"
	StateCookie   = "tp_oauth"
	SessionTTL    = 30 * 24 * time.Hour
	InviteTTL     = 7 * 24 * time.Hour
	stateTTL      = 10 * time.Minute
)

// Config configures the auth service.
type Config struct {
	AppBaseURL   string // web app origin, used for invite URLs and redirects
	CookieSecure bool
	DevLogin     bool
}

// Service implements authentication use cases.
type Service struct {
	pool *pgxpool.Pool
	q    *store.Queries
	cfg  Config
	idp  IdentityProvider
	now  func() time.Time
}

// New builds a Service. idp may be nil when Google login is not configured.
func New(pool *pgxpool.Pool, cfg Config, idp IdentityProvider) *Service {
	return &Service{pool: pool, q: store.New(pool), cfg: cfg, idp: idp, now: time.Now}
}

// Config returns the configuration.
func (s *Service) Config() Config { return s.cfg }

// NewToken returns a random URL-safe token (256 bits).
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the SHA-256 of a token (what is stored in the database).
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// Authenticate resolves a session token to its Learner.
func (s *Service) Authenticate(ctx context.Context, token string) (store.Learner, error) {
	if token == "" {
		return store.Learner{}, app.E(app.KindUnauthorized, "unauthenticated", "login required")
	}
	l, err := s.q.GetSessionLearner(ctx, HashToken(token))
	if store.IsNotFound(err) {
		return l, app.E(app.KindUnauthorized, "unauthenticated", "session expired or invalid")
	}
	return l, err
}

// StartSession creates a session for a Learner and returns its cookie.
func (s *Service) StartSession(ctx context.Context, learnerID uuid.UUID) (*http.Cookie, error) {
	token, err := NewToken()
	if err != nil {
		return nil, err
	}
	exp := s.now().Add(SessionTTL)
	if err := s.q.CreateSession(ctx, store.CreateSessionParams{TokenHash: HashToken(token), LearnerID: learnerID, ExpiresAt: exp}); err != nil {
		return nil, err
	}
	return &http.Cookie{ //nolint:gosec // Secure is configurable (SESSION_COOKIE_SECURE) for local http dev
		Name: SessionCookie, Value: token, Path: "/", Expires: exp, MaxAge: int(SessionTTL.Seconds()),
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	}, nil
}

// EndSession deletes a session and returns a clearing cookie.
func (s *Service) EndSession(ctx context.Context, token string) (*http.Cookie, error) {
	if token != "" {
		if err := s.q.DeleteSession(ctx, HashToken(token)); err != nil {
			return nil, err
		}
	}
	return &http.Cookie{ //nolint:gosec // Secure is configurable (SESSION_COOKIE_SECURE) for local http dev
		Name: SessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	}, nil
}

// PurgeExpiredSessions deletes expired sessions.
func (s *Service) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredSessions(ctx)
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", app.E(app.KindBadRequest, "invalid_email", "invalid email address")
	}
	return email, nil
}

// DevLogin logs in by email without Google (AUTH_DEV_LOGIN=true only). It
// creates the Learner if missing, without an invite; the first-ever Learner
// becomes the Owner.
func (s *Service) DevLogin(ctx context.Context, email, name string) (store.Learner, error) {
	if !s.cfg.DevLogin {
		return store.Learner{}, app.E(app.KindNotFound, "not_found", "dev login is disabled")
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return store.Learner{}, err
	}
	return s.resolveLogin(ctx, email, name, "", true)
}

// resolveLogin finds or creates the Learner for a verified identity.
func (s *Service) resolveLogin(ctx context.Context, email, name, inviteToken string, skipInvite bool) (store.Learner, error) {
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	var learner store.Learner
	err := store.InTx(ctx, s.pool, func(q *store.Queries) error {
		l, err := q.GetLearnerByEmail(ctx, email)
		if err == nil {
			learner = l
			return nil
		}
		if !store.IsNotFound(err) {
			return err
		}
		n, err := q.CountLearners(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			// First-ever login becomes the Owner. The single-owner unique index
			// makes a concurrent second "first" login fail instead of creating two Owners.
			learner, err = q.CreateLearner(ctx, store.CreateLearnerParams{Email: email, Name: name, IsOwner: true})
			return err
		}
		if skipInvite {
			learner, err = q.CreateLearner(ctx, store.CreateLearnerParams{Email: email, Name: name})
			return err
		}
		if inviteToken == "" {
			return app.E(app.KindForbidden, "invite_required", "this platform is invite-only")
		}
		inv, err := q.GetInviteByHashForUpdate(ctx, HashToken(inviteToken))
		if store.IsNotFound(err) {
			return app.E(app.KindForbidden, "invite_invalid", "invite is invalid")
		}
		if err != nil {
			return err
		}
		if inv.UsedBy != nil || !s.now().Before(inv.ExpiresAt) {
			return app.E(app.KindForbidden, "invite_invalid", "invite was already used or has expired")
		}
		learner, err = q.CreateLearner(ctx, store.CreateLearnerParams{Email: email, Name: name})
		if err != nil {
			return err
		}
		return q.UseInvite(ctx, store.UseInviteParams{ID: inv.ID, UsedBy: &learner.ID})
	})
	if err != nil && store.IsUniqueViolation(err) {
		return store.Learner{}, app.E(app.KindConflict, "login_conflict", "concurrent login; please retry")
	}
	return learner, err
}

// InviteView is an invite (the token is only available at creation).
type InviteView struct {
	ID        uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedBy    *string
	URL       string
}

// CreateInvite creates an invite (Owner only) and returns its one-time URL.
func (s *Service) CreateInvite(ctx context.Context, a app.Actor) (InviteView, error) {
	if !a.IsOwner {
		return InviteView{}, app.E(app.KindForbidden, "forbidden", "only the Owner may invite")
	}
	token, err := NewToken()
	if err != nil {
		return InviteView{}, err
	}
	inv, err := s.q.CreateInvite(ctx, store.CreateInviteParams{TokenHash: HashToken(token), CreatedBy: a.LearnerID, ExpiresAt: s.now().Add(InviteTTL)})
	if err != nil {
		return InviteView{}, err
	}
	return InviteView{
		ID: inv.ID, CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
		URL: strings.TrimRight(s.cfg.AppBaseURL, "/") + "/login?invite=" + token,
	}, nil
}

// ListInvites lists invites (Owner only).
func (s *Service) ListInvites(ctx context.Context, a app.Actor) ([]InviteView, error) {
	if !a.IsOwner {
		return nil, app.E(app.KindForbidden, "forbidden", "only the Owner may list invites")
	}
	rows, err := s.q.ListInvites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InviteView, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, InviteView{ID: r.ID, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, UsedBy: r.UsedByEmail})
	}
	return out, nil
}

// ErrNotConfigured means Google login is not configured.
var ErrNotConfigured = errors.New("auth: google login not configured")
