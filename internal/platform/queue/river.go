package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"e-com-rental-cloth/backend/internal/platform/db/dbsqlc"
)

type PurgeStaleSessionsArgs struct{}

func (PurgeStaleSessionsArgs) Kind() string { return "platform:purge_stale_sessions" }

type PurgeStaleSessionsWorker struct {
	river.WorkerDefaults[PurgeStaleSessionsArgs]
	queries *dbsqlc.Queries
}

func NewPurgeStaleSessionsWorker(pool *pgxpool.Pool) *PurgeStaleSessionsWorker {
	return &PurgeStaleSessionsWorker{queries: dbsqlc.New(pool)}
}

func (w *PurgeStaleSessionsWorker) Work(ctx context.Context, _ *river.Job[PurgeStaleSessionsArgs]) error {
	deleted, err := w.queries.PurgeStaleSessions(ctx)
	if err != nil {
		return err
	}
	if deleted > 0 {
		slog.InfoContext(ctx, "purged stale sessions", slog.Int64("deleted_count", deleted))
	}
	return nil
}

// NewInsertOnlyClient creates a River client for the HTTP API process to enqueue jobs
// transactionally (via InsertTx) without running background workers inside the API server.
func NewInsertOnlyClient(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

// NewWorkerClient creates a River client that actively processes background queues in cmd/worker.
func NewWorkerClient(pool *pgxpool.Pool, workers *river.Workers) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return PurgeStaleSessionsArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
		},
		Workers: workers,
	})
}
