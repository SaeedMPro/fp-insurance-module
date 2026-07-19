// Command seed loads the demo fixtures (internal/fixtures) through the real
// service layer, so seeded data carries genuine audit entries and pricing.
// Idempotent — safe to run repeatedly.
package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"insurance-module/internal/app"
	"insurance-module/internal/fixtures"
	"insurance-module/internal/platform/config"
	"insurance-module/internal/storage/postgres"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	store, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	services := app.Build(store, app.Options{JWTSecret: cfg.JWTSecret, JWTTTL: cfg.JWTTTL})

	if err := fixtures.Seed(context.Background(), store, services); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println(fixtures.DemoAccounts)
}
