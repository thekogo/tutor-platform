// Package mcpserver exposes the Content Admin operations as MCP tools
// (github.com/modelcontextprotocol/go-sdk) over streamable HTTP at /mcp,
// authenticated with a bearer token (MCP_TOKEN). The caller acts as the
// Content Admin with Owner visibility (ADR 0003). Tools are thin: validation
// and business rules live in the app layer.
package mcpserver

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thekogo/tutor-platform/backend/internal/app"
)

// Version is reported to MCP clients.
const Version = "0.1.0"

// NewServer builds the MCP server with every Content Admin tool registered.
func NewServer(svc *app.Service) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "tutor-platform", Version: Version}, &mcp.ServerOptions{
		Instructions: "Content Admin tools for the tutor platform. Courses, Topics and Concepts are addressed by slug. " +
			"Questions are created as Draft; an independent verifier (who must not see the key) answers them via " +
			"list_questions_for_verification + submit_verification, and only matching Questions become Active.",
	})
	t := &tools{app: svc}
	t.register(s)
	return s
}

// Handler returns the /mcp HTTP handler with bearer-token auth. An empty
// token disables the endpoint (503).
func Handler(svc *app.Service, token string, log *slog.Logger) http.Handler {
	server := NewServer(svc)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       log,
	})
	want := sha256.Sum256([]byte(token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			writeJSONError(w, http.StatusServiceUnavailable, "mcp_disabled", "MCP_TOKEN is not configured")
			return
		}
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		gotSum := sha256.Sum256([]byte(got))
		if !ok || subtle.ConstantTimeCompare(gotSum[:], want[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
			writeJSONError(w, http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}
