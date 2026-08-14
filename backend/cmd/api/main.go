// Command api is the Supplementary Insurance Module's HTTP server. It is the
// composition boundary: config → schema init → store → services (internal/app)
// → REST router, served with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"insurance-module/internal/app"
	"insurance-module/internal/platform/config"
	"insurance-module/internal/platform/database"
	"insurance-module/internal/platform/logging"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger := logging.Setup(cfg.IsProduction())

	if err := database.InitSchema(context.Background(), cfg.DatabaseURL, cfg.DBInitPath); err != nil {
		logger.Error("schema init failed", "error", err)
		os.Exit(1)
	}

	store, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}

	services := app.Build(store, app.Options{
		JWTSecret: cfg.JWTSecret,
		JWTTTL:    cfg.JWTTTL,
	})

	router := transporthttp.NewRouter(transporthttp.Config{
		JWTSecret:   cfg.JWTSecret,
		CORSOrigins: cfg.CORSOrigins,
		Logger:      logger,
	}, services)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("api listening", "port", cfg.HTTPPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Warn("graceful shutdown failed", "error", err)
	}
}
