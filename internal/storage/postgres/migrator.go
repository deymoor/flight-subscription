package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsAdvisoryLockKey int64 = 4972134809123451

type migration struct {
	version string
	path    string
	sql     string
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationsAdvisoryLockKey); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationsAdvisoryLockKey)
	}()

	if err := ensureMigrationTable(ctx, pool); err != nil {
		return err
	}

	migrations, err := readMigrations(dir)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := isMigrationApplied(ctx, pool, migration.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applyMigration(ctx, pool, migration); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	)

	return err
}

func readMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		version := strings.TrimSuffix(entry.Name(), ".up.sql")
		sql := strings.TrimSpace(string(data))
		if sql == "" {
			return nil, fmt.Errorf("empty migration %s", path)
		}

		migrations = append(migrations, migration{
			version: version,
			path:    path,
			sql:     sql,
		})
	}

	return migrations, nil
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var applied bool
	err := pool.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
		version,
	).Scan(&applied)
	if err != nil {
		return false, err
	}

	return applied, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, migration migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, migration.sql, pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("apply migration %s: %w", migration.path, err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`,
		migration.version,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
