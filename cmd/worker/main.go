package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverqueue/river"

	"e-com-rental-cloth/backend/internal/platform/config"
	"e-com-rental-cloth/backend/internal/platform/db"
	"e-com-rental-cloth/backend/internal/platform/queue"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, queue.NewPurgeStaleSessionsWorker(pool))
	// Register additional domain background workers here (e.g. notification email/LINE, booking expiration cleanup)

	riverClient, err := queue.NewWorkerClient(pool, workers)
	if err != nil {
		slog.Error("failed to initialize River worker client", slog.Any("err", err))
		os.Exit(1)
	}

	if err := riverClient.Start(ctx); err != nil {
		slog.Error("failed to start River worker", slog.Any("err", err))
		os.Exit(1)
	}
	slog.Info("started background queue worker", slog.String("env", cfg.AppEnv))

	<-ctx.Done()
	slog.Info("shutting down background queue worker")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = riverClient.Stop(shutdownCtx)
}
