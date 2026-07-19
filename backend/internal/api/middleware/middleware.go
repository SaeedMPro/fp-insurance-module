// Package middleware provides authentication (JWT for users, API key for the
// parent-system integration) and role-based access control for the REST API.
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"insurance-module/internal/auth"
	"insurance-module/internal/models"
)

type contextKey int

const (
	userClaimsKey contextKey = iota
)

// Authenticate verifies the Bearer JWT on every request and stores its claims
// in the request context. It rejects the request outright if the token is
// missing or invalid; per-route role checks are handled separately by RequireRole.
func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httpError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				httpError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the authenticated user's claims set by Authenticate.
func UserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*auth.Claims)
	return claims, ok
}

// RequireRole restricts a route to the given roles. It must run after Authenticate.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := UserFromContext(r.Context())
			if !ok {
				httpError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if !allowed[claims.Role] {
				httpError(w, http.StatusForbidden, "insufficient role for this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAPIKey authenticates the parent-system integration endpoints via a
// static API key (header "X-API-Key"), independent of the JWT user session.
func RequireAPIKey(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				httpError(w, http.StatusUnauthorized, "missing X-API-Key header")
				return
			}
			hash := sha256Hex(key)
			var count int64
			if err := db.WithContext(r.Context()).Model(&models.IntegrationAPIKey{}).
				Where("api_key_hash = ? AND is_active = true", hash).Count(&count).Error; err != nil || count == 0 {
				httpError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func httpError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
