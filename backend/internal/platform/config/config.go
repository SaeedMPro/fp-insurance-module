// Package config loads runtime configuration from environment variables and
// validates it: a production process must never come up with development
// fallbacks for secrets.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// insecureDefaultJWTSecret is the development fallback. Load refuses to return
// a production config that still carries it — it exists precisely so it can be
// recognised and rejected, not to protect anything.
const insecureDefaultJWTSecret = "dev-only-insecure-secret-change-me" // #nosec G101 -- known-insecure sentinel, refused in production

type Config struct {
	Env         string
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	DBInitPath  string
	CORSOrigins []string
	// AttachmentsDir is where uploaded claim documents are stored. In Docker
	// this should be a mounted volume; otherwise uploads vanish when the
	// container is replaced.
	AttachmentsDir string
}

// IsProduction reports whether the process runs with APP_ENV=production.
func (c Config) IsProduction() bool { return c.Env == "production" }

// Load reads configuration from the environment and validates it.
func Load() (Config, error) {
	cfg := Config{
		Env:            getEnv("APP_ENV", "development"),
		HTTPPort:       getEnv("HTTP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://insurance:insurance@localhost:5432/insurance?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", insecureDefaultJWTSecret),
		JWTTTL:         getDuration("JWT_TTL", 8*time.Hour),
		DBInitPath:     getEnv("DB_INIT_PATH", "db/init.sql"),
		AttachmentsDir: getEnv("ATTACHMENTS_DIR", "data/attachments"),
		CORSOrigins:    []string{getEnv("CORS_ORIGIN", "http://localhost:5173")},
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var errs []error
	if c.IsProduction() {
		if c.JWTSecret == insecureDefaultJWTSecret || c.JWTSecret == "" {
			errs = append(errs, errors.New("JWT_SECRET must be set to a real secret in production"))
		}
		if len(c.JWTSecret) > 0 && len(c.JWTSecret) < 16 {
			errs = append(errs, errors.New("JWT_SECRET is too short (min 16 bytes)"))
		}
		if os.Getenv("DATABASE_URL") == "" {
			errs = append(errs, errors.New("DATABASE_URL must be set explicitly in production"))
		}
	}
	if c.JWTTTL <= 0 {
		errs = append(errs, fmt.Errorf("JWT_TTL must be positive, got %s", c.JWTTTL))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
