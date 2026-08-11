package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tax-client/backend/internal/httpapi"
	"tax-client/backend/internal/repository"
	"tax-client/backend/internal/seed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := env("DATABASE_URL", "postgres://taxapp:taxapp@localhost:55432/taxdb?sslmode=disable")
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer migrateCancel()
		if err := repository.RunPostgresMigrations(migrateCtx, databaseURL); err != nil {
			logger.Error("migrate postgres", "error", err)
			os.Exit(1)
		}
		logger.Info("postgres migrations completed")
		return
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 15*time.Second)
	repo, err := repository.NewPostgresRepository(connectCtx, databaseURL, seed.Database())
	connectCancel()
	if err != nil {
		logger.Error("initialize repository", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	handler := httpapi.New(repo, httpapi.Config{
		AllowedOrigin: env("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
		Token:         env("AUTH_TOKEN", "mock-session-token"),
		Logger:        logger,
	})
	server := &http.Server{
		Addr:              httpAddress(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("tax API started", "address", server.Addr, "storage", "postgresql")
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

func httpAddress() string {
	if address := os.Getenv("HTTP_ADDR"); address != "" {
		return address
	}
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return ":" + port
	}
	return ":8080"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
