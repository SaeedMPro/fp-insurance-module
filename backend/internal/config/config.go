// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"time"
)

type Config struct {
	Env            string
	HTTPPort       string
	DatabaseURL    string
	JWTSecret      string
	JWTTTL         time.Duration
	MigrationsPath string
	CORSOrigins    []string
}

func Load() Config {
	return Config{
		Env:            getEnv("APP_ENV", "development"),
		HTTPPort:       getEnv("HTTP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://insurance:insurance@localhost:5432/insurance?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-only-insecure-secret-change-me"),
		JWTTTL:         getDuration("JWT_TTL", 8*time.Hour),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "file://migrations"),
		CORSOrigins:    []string{getEnv("CORS_ORIGIN", "http://localhost:5173")},
	}
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
