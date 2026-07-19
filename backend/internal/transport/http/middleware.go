package http

import (
	"context"
	"net/http"
	"strings"

	"insurance-module/internal/auth"
	"insurance-module/internal/domain"
)

type contextKey int

const actorKey contextKey = iota

// authenticate verifies the Bearer JWT and stores the resulting domain.Actor
// in the request context. Per-route role checks are requireRole's job.
func authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			claims, err := auth.ParseToken(secret, strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), actorKey, claims.Actor())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// actorFrom retrieves the authenticated actor set by authenticate.
func actorFrom(r *http.Request) (domain.Actor, bool) {
	actor, ok := r.Context().Value(actorKey).(domain.Actor)
	return actor, ok
}

// mustActor is for handlers behind authenticate: the actor is always present.
func mustActor(w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	actor, ok := actorFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
	}
	return actor, ok
}

// requireRole restricts a route to the given roles; must follow authenticate.
func requireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := make(map[domain.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := actorFrom(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if !allowed[actor.Role] {
				writeError(w, http.StatusForbidden, "insufficient role for this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyVerifier is the slice of the integration service the middleware needs.
type apiKeyVerifier interface {
	VerifyAPIKey(ctx context.Context, key string) (bool, error)
}

// requireAPIKey authenticates parent-system endpoints via the X-API-Key
// header, independent of the JWT user session.
func requireAPIKey(verifier apiKeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				writeError(w, http.StatusUnauthorized, "missing X-API-Key header")
				return
			}
			ok, err := verifier.VerifyAPIKey(r.Context(), key)
			if err != nil || !ok {
				writeError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
