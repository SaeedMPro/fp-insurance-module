// Command api is the Supplementary Insurance Module's HTTP server: it wires
// config, the database connection, schema migrations, the domain services
// (rule engine, workflow engine, audit trail), and the REST API router, then
// serves it with a graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"insurance-module/internal/api"
	"insurance-module/internal/audit"
	"insurance-module/internal/config"
	"insurance-module/internal/db"
	"insurance-module/internal/reports"
	"insurance-module/internal/ruleengine"
	"insurance-module/internal/workflow"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if err := db.Migrate(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	gdb, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	rulesEngine := ruleengine.NewEngine(gdb)
	auditSvc := audit.NewService(gdb)
	workflowEngine := workflow.NewEngine(gdb, rulesEngine, auditSvc)
	reportsSvc := reports.NewService(gdb)

	router := api.NewRouter(api.Deps{
		DB:       gdb,
		Cfg:      cfg,
		Rules:    rulesEngine,
		Workflow: workflowEngine,
		Audit:    auditSvc,
		Reports:  reportsSvc,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("insurance-module api listening on :%s (env=%s)", cfg.HTTPPort, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
