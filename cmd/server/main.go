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

	"tax-client/backend/internal/httpapi"
	"tax-client/backend/internal/repository"
	"tax-client/backend/internal/seed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	dataPath := env("TAX_DATA_PATH", "./data/tax.json")
	repo, err := repository.NewJSONRepository(dataPath, seed.Database())
	if err != nil {
		logger.Error("initialize repository", "error", err)
		os.Exit(1)
	}

	handler := httpapi.New(repo, httpapi.Config{
		AllowedOrigin: env("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
		Token:         env("AUTH_TOKEN", "mock-session-token"),
		Logger:        logger,
	})
	server := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("tax API started", "address", server.Addr, "data", dataPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve API", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown API", "error", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
