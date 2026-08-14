// Command createadmin bootstraps the sole admin account when none exists.
// Prefer make create-admin; seed.sql also inserts the demo admin.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"

	"insurance-module/internal/app"
	"insurance-module/internal/platform/config"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
)

func main() {
	_ = godotenv.Load()

	username := flag.String("username", envOr("ADMIN_USERNAME", "admin"), "admin username")
	password := flag.String("password", envOr("ADMIN_PASSWORD", "Admin123!"), "admin password")
	fullName := flag.String("full-name", envOr("ADMIN_FULL_NAME", "مدیر سامانه"), "admin display name")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	store, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	svcs := app.Build(store, app.Options{JWTSecret: cfg.JWTSecret, JWTTTL: cfg.JWTTTL})
	user, created, err := svcs.Users.EnsureAdmin(context.Background(), users.CreateInput{
		Username: *username,
		Password: *password,
		FullName: *fullName,
	})
	if err != nil {
		log.Fatalf("ensure admin: %v", err)
	}
	if created {
		log.Printf("admin created: %s (%s)", user.Username, user.FullName)
		return
	}
	log.Printf("admin already exists: %s (%s) — no changes", user.Username, user.FullName)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
