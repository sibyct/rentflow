// Command migrate applies pending schema migrations and exits — no
// server, no long-running process. It exists so `flyctl deploy`'s
// release_command can run migrations from a machine on the app's own
// private network, without needing the production database reachable
// from anywhere else (a GitHub-hosted CI runner included). Because
// release_command executes the already-built image directly rather
// than through a shell, this reads DATABASE_URL via os.Getenv instead
// of relying on shell interpolation — see fly.toml/fly.staging.toml's
// [deploy] release_command line and backend/Dockerfile for how this
// binary gets built and invoked.
//
// Uses the pgx-based driver (matching the rest of this codebase)
// rather than golang-migrate's default lib/pq driver, so there's no
// second Postgres driver dependency to maintain.
package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsSource = "file:///app/migrations"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	migratorURL, err := toPgx5Scheme(databaseURL)
	if err != nil {
		return fmt.Errorf("parsing DATABASE_URL: %w", err)
	}

	m, err := migrate.New(migrationsSource, migratorURL)
	if err != nil {
		return fmt.Errorf("connecting migrator: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}

	fmt.Println("migrations applied (or already up to date)")
	return nil
}

// toPgx5Scheme rewrites DATABASE_URL's scheme from postgres(ql):// to
// pgx5:// — the scheme golang-migrate's pgx/v5 driver package
// registers itself under (database/pgx/v5/pgx.go: database.Register
// ("pgx5", &db)). DATABASE_URL itself stays a standard postgres:// DSN
// everywhere else in this app (the api binary's own pgx pool,
// docker-compose, fly secrets) — this translation is local to this
// one binary, not a change to that shared convention.
func toPgx5Scheme(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	u.Scheme = "pgx5"
	return u.String(), nil
}
