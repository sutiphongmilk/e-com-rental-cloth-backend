package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"e-com-rental-cloth/backend/internal/platform/config"
	"e-com-rental-cloth/backend/internal/platform/db"
	"e-com-rental-cloth/backend/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database for migration", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	// 1. Apply River queue schema migrations
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		slog.Error("failed to create river migrator", slog.Any("err", err))
		os.Exit(1)
	}
	if _, err := riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		slog.Error("failed to apply river migrations", slog.Any("err", err))
		os.Exit(1)
	}

	// 2. Apply application Goose SQL migrations
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	if err := migrations.Up(ctx, sqlDB); err != nil {
		slog.Error("failed to apply application sql migrations", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("all database and queue migrations applied successfully", slog.String("env", cfg.AppEnv))
}
