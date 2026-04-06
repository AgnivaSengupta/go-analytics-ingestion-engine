package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockKey int64 = 73422191

func main() {
	dbURL := os.Getenv("DB_DSN")
	if strings.TrimSpace(dbURL) == "" {
		dbURL = "postgres://postgres:password@localhost:5432/analytics?sslmode=disable"
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if strings.TrimSpace(migrationsDir) == "" {
		migrationsDir = filepath.Join(".", "infra", "migrations")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := runMigrations(ctx, dbPool, migrationsDir); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations applied successfully")
}

func runMigrations(ctx context.Context, db *pgxpool.Pool, migrationsDir string) error {
	if err := ensureMetadata(ctx, db); err != nil {
		return err
	}

	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for advisory lock: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey); unlockErr != nil {
			log.Printf("warning: failed to release advisory lock: %v", unlockErr)
		}
	}()

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}

	migrationFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			migrationFiles = append(migrationFiles, name)
		}
	}
	slices.Sort(migrationFiles)

	for _, filename := range migrationFiles {
		applied, err := isApplied(ctx, db, filename)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(migrationsDir, filename)
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", filename, err)
		}
		if strings.TrimSpace(string(contents)) == "" {
			return fmt.Errorf("migration %q is empty", filename)
		}

		log.Printf("applying migration %s", filename)
		if err := applyMigration(ctx, db, filename, string(contents)); err != nil {
			return err
		}
	}

	return nil
}

func ensureMetadata(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func isApplied(ctx context.Context, db *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE version = $1
		)
	`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %q: %w", version, err)
	}
	return exists, nil
}

func applyMigration(ctx context.Context, db *pgxpool.Pool, version, sql string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", version, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()

	if _, err = tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("execute migration %q: %w", version, err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO schema_migrations (version)
		VALUES ($1)
	`, version); err != nil {
		return fmt.Errorf("record migration %q: %w", version, err)
	}

	if err = tx.Commit(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("commit migration %q: %w", version, err)
		}
		return fmt.Errorf("commit migration %q: %w", version, err)
	}

	return nil
}
