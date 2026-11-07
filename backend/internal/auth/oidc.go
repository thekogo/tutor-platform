package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Identity is a verified external identity.
type Identity struct {
	Email         string
	EmailVerified bool
	Name          string
}

// IdentityProvider abstracts the OIDC provider (Google in production, a fake in tests).
type IdentityProvider interface {
	AuthCodeURL(ctx context.Context, state, nonce, verifier string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error)
}

// GoogleConfig configures Google OIDC.
type GoogleConfig struct {
	ClientID, ClientSecret, RedirectURL string
}

// Google is an IdentityProvider backed by accounts.google.com. Provider
// discovery is lazy so the server starts without network access.
type Google struct {
	cfg      GoogleConfig
	mu       sync.Mutex
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewGoogle returns a Google IdentityProvider, or nil if not configured.
func NewGoogle(cfg GoogleConfig) *Google {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return nil
	}
	return &Google{cfg: cfg}
}

func (g *Google) init(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.oauth != nil {
		return nil
	}
	p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return fmt.Errorf("auth: oidc discovery: %w", err)
	}
	g.oauth = &oauth2.Config{
		ClientID: g.cfg.ClientID, ClientSecret: g.cfg.ClientSecret, RedirectURL: g.cfg.RedirectURL,
		Endpoint: p.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
	}
	g.verifier = p.Verifier(&oidc.Config{ClientID: g.cfg.ClientID})
	return nil
}

// AuthCodeURL builds the Google authorization URL (with PKCE and nonce).
func (g *Google) AuthCodeURL(ctx context.Context, state, nonce, verifier string) string {
	if err := g.init(ctx); err != nil {
		return ""
	}
	return g.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
}

// Exchange trades the code for tokens and verifies the ID token.
func (g *Google) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {
	if err := g.init(ctx); err != nil {
		return Identity{}, err
	}
	tok, err := g.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, fmt.Errorf("auth: code exchange: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return Identity{}, errors.New("auth: no id_token in token response")
	}
	idt, err := g.verifier.Verify(ctx, raw)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idt.Nonce), []byte(nonce)) != 1 {
		return Identity{}, errors.New("auth: nonce mismatch")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idt.Claims(&claims); err != nil {
		return Identity{}, err
	}
	return Identity{Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name}, nil
}

// loginState is kept in a short-lived HttpOnly cookie between login and callback.
type loginState struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Invite   string `json:"i,omitempty"`
}

// BeginLogin returns the provider redirect URL and the state cookie to set.
func (s *Service) BeginLogin(ctx context.Context, invite string) (string, *http.Cookie, error) {
	if s.idp == nil {
		return "", nil, app.E(app.KindUnavailable, "login_unavailable", "Google login is not configured")
	}
	st := loginState{Invite: invite}
	var err error
	if st.State, err = NewToken(); err != nil {
		return "", nil, err
	}
	if st.Nonce, err = NewToken(); err != nil {
		return "", nil, err
	}
	st.Verifier = oauth2.GenerateVerifier()
	u := s.idp.AuthCodeURL(ctx, st.State, st.Nonce, st.Verifier)
	if u == "" {
		return "", nil, app.E(app.KindUnavailable, "login_unavailable", "identity provider unreachable")
	}
	b, err := json.Marshal(st)
	if err != nil {
		return "", nil, err
	}
	c := &http.Cookie{ //nolint:gosec // Secure is configurable (SESSION_COOKIE_SECURE) for local http dev
		Name: StateCookie, Value: base64.RawURLEncoding.EncodeToString(b), Path: "/api/auth/google",
		MaxAge: int(stateTTL.Seconds()), HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	}
	return u, c, nil
}

// ClearStateCookie expires the login state cookie.
func (s *Service) ClearStateCookie() *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure is configurable (SESSION_COOKIE_SECURE) for local http dev
		Name: StateCookie, Value: "", Path: "/api/auth/google", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode}
}

// CompleteLogin validates the callback against the state cookie, verifies the
// identity, and resolves (or creates) the Learner.
func (s *Service) CompleteLogin(ctx context.Context, code, state, stateCookie string) (store.Learner, error) {
	if s.idp == nil {
		return store.Learner{}, app.E(app.KindUnavailable, "login_unavailable", "Google login is not configured")
	}
	var st loginState
	raw, err := base64.RawURLEncoding.DecodeString(stateCookie)
	if err != nil || json.Unmarshal(raw, &st) != nil || st.State == "" ||
		subtle.ConstantTimeCompare([]byte(st.State), []byte(state)) != 1 {
		return store.Learner{}, app.E(app.KindForbidden, "invalid_state", "login state mismatch; please try again")
	}
	id, err := s.idp.Exchange(ctx, code, st.Verifier, st.Nonce)
	if err != nil {
		return store.Learner{}, app.E(app.KindForbidden, "login_failed", "could not verify your Google account")
	}
	if !id.EmailVerified {
		return store.Learner{}, app.E(app.KindForbidden, "email_unverified", "your Google email is not verified")
	}
	email, err := normalizeEmail(id.Email)
	if err != nil {
		return store.Learner{}, err
	}
	return s.resolveLogin(ctx, email, id.Name, st.Invite, false)
}
