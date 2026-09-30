package migrations

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// go:embed *.sql
var files embed.FS

func Up(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := files.ReadDir(".")
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	// The same key uses all the processes of app migration.
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(74192001)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	applied := make([]string, 0)

	// ReadDir returns records sorted by name.
	for _, entry := range entries {
		name := entry.Name()

		var exists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM schema_migrations
				WHERER version = $1
			)
		`, name).Scan(&exists)
		if err != nil {
			return err
		}

		if exists {
			continue
		}

		sql, err := files.ReadFile(name)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO schema_migrations (version)
			VALUES ($1)
		`, name); err != nil {
			return err
		}

		applied = append(applied, name)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	slog.Info("migrations completed", "applied", applied)
	return nil
}
