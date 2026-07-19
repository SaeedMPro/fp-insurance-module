// Package auth is a small leaf utility: password hashing and JWT
// issuance/verification. It depends only on domain types; token policy
// (who gets one, TTL, secret) belongs to the users service and config.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"insurance-module/internal/domain"
)

var ErrInvalidToken = errors.New("auth: invalid or expired token")

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// TokenSubject is the identity a token is issued for.
type TokenSubject struct {
	UserID   uuid.UUID
	Username string
	Role     domain.Role
}

// Claims are the JWT claims embedded in every access token.
type Claims struct {
	UserID   uuid.UUID   `json:"user_id"`
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// Actor converts verified claims into the domain actor used by services.
func (c *Claims) Actor() domain.Actor {
	return domain.Actor{UserID: c.UserID, Username: c.Username, Role: c.Role}
}

func GenerateToken(secret string, ttl time.Duration, sub TokenSubject) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   sub.UserID,
		Username: sub.Username,
		Role:     sub.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   sub.UserID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
