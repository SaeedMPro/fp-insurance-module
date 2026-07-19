package config

import (
	"strings"
	"testing"
)

// setProdEnv configures a minimal valid production environment, then lets each
// test poke holes in it.
func setProdEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "a-real-secret-of-decent-length")
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/insurance")
	t.Setenv("JWT_TTL", "")
}

func TestLoad_DevelopmentDefaultsAreAccepted(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATABASE_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("dev defaults must load, got: %v", err)
	}
	if cfg.IsProduction() {
		t.Fatal("empty APP_ENV must not be production")
	}
}

func TestLoad_ProductionRefusesDefaultSecret(t *testing.T) {
	setProdEnv(t)
	t.Setenv("JWT_SECRET", insecureDefaultJWTSecret)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("want JWT_SECRET error, got: %v", err)
	}
}

func TestLoad_ProductionRefusesShortSecret(t *testing.T) {
	setProdEnv(t)
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "too short") {
		t.Fatalf("want too-short error, got: %v", err)
	}
}

func TestLoad_ProductionRequiresExplicitDatabaseURL(t *testing.T) {
	setProdEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("want DATABASE_URL error, got: %v", err)
	}
}

func TestLoad_ProductionHappyPath(t *testing.T) {
	setProdEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("valid production env must load, got: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production config")
	}
}

func TestLoad_RejectsNonPositiveTTL(t *testing.T) {
	setProdEnv(t)
	t.Setenv("JWT_TTL", "-1h")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JWT_TTL") {
		t.Fatalf("want JWT_TTL error, got: %v", err)
	}
}
