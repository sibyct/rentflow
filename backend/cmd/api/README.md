# `cmd/api` — service bootstrap

This package is the **only** place in the codebase allowed to know about
every layer at once. It has two jobs: wire the dependency graph, and run
the process until it's told to stop. It contains no business logic.

## `main()` / `run()` split

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error { /* everything else */ }
```

`main()` is the *only* place `os.Exit` is called, and it's called exactly
once. Everything else — config, logger, connections, wiring, the server,
shutdown — lives in `run()` and returns a single wrapped `error`. This
makes the whole bootstrap sequence testable in principle (you can call
`run()` from a test with a canceled context) and keeps `main()` too small
to have bugs in.

`run()` takes no arguments and closes over nothing at package scope —
config, the logger, the DB pool, and the cache client are all local
variables constructed inside it and passed explicitly to everything that
needs them. There is no `var db *pgxpool.Pool` or `var log *slog.Logger`
at package level, and `slog.SetDefault` is deliberately **not** called —
every constructor below receives `log` as an argument.

## Order of operations inside `run()`

```
1. config.Load()                    → fail fast on missing/invalid env vars
2. build the slog.Logger             → passed explicitly from here on
3. signal.NotifyContext(SIGINT/TERM) → one root context for the process
4. connect to Postgres  (5s timeout, ping-verified, error wrapped)
5. connect to Redis     (5s timeout, ping-verified, non-fatal — see below)
6. construct repositories   (postgres.PropertyRepository, postgres.UserRepository)
7. construct services       (service.PropertyService, service.AuthService)
8. construct handlers       (handlers.AuthHandler, HealthHandler, VersionHandler)
9. build the router (middleware: request ID → logger → recover → access log → CORS → auth)
10. build *http.Server with explicit Read/Write/Idle timeouts
11. srv.ListenAndServe() in a goroutine; http.ErrServerClosed is not an error
12. select on the signal context vs. a server-error channel
13. on shutdown: srv.Shutdown(ctx) with a 10s bound, then close DB/cache via defer
```

Steps 4–10 are a straight line: each layer is constructed from the
concrete thing below it and handed to the layer above it as a **domain
interface**, never a concrete type. `internal/transport/http/router.go`
depends on `domain.AuthService` and `domain.PropertyService`, not on
`*service.AuthService` — so this file is the only place that ever
imports both `internal/service` and `internal/transport/http` at once.

## Error wrapping

Every error `run()` returns up to `main()` is wrapped with `fmt.Errorf("doing
X: %w", err)` — `loading config: %w`, `connecting to database: %w`,
`shutting down http server: %w`. Nothing is ever returned bare. `errors.Is`
still works through the chain (e.g. `errors.Is(err, http.ErrServerClosed)`
in the listener goroutine), and the top-level message on stderr always
tells you which step failed without needing a debugger.

## Graceful shutdown

`internal/transport/http/server.go` holds two small helpers:

- `NewServer(addr, handler)` — builds `*http.Server` with `ReadHeaderTimeout`,
  `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` all set explicitly (never
  left at Go's zero-value defaults, which effectively means "no timeout").
- `Shutdown(ctx, srv, timeout)` — wraps `srv.Shutdown` in a bounded
  `context.WithTimeout` and error-wraps the result. This is the "graceful
  shutdown helper" referenced in the header comment above.

`run()` calls `Shutdown(context.Background(), srv, 10*time.Second)` once
the signal context is done, then lets its `defer pool.Close()` and
`defer cache.Close()` run as `run()` returns — connections are always
closed, whether the process exits via a clean shutdown or an early error.

## Version / build metadata

```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)
```

These are package-level `var`s, which looks like it contradicts "no
package-level mutable state" — but they're categorically different from
a shared `*sql.DB` or logger global: nothing in the program ever
reassigns them at runtime. They're baked in **before `main()` runs**, via
the linker:

```bash
make build   # wires this up automatically, see Makefile
```

```makefile
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)
```

`go build -ldflags "$(LDFLAGS)" ./cmd/api`. `go run` and a plain `go build`
(without `make`) fall back to `"dev"` / `"unknown"`, which is fine for
local development.

## Endpoints registered by this wiring

| Method | Path        | Purpose                                                                 | Auth |
|--------|-------------|--------------------------------------------------------------------------|------|
| GET    | `/healthz`  | Liveness — process is up. Never checks dependencies.                    | none |
| GET    | `/readyz`   | Readiness — pings Postgres (and Redis if configured). 503 if unreachable. | none |
| GET    | `/version`  | Returns `{version, commit, build_date}` from the ldflags above.          | none |

All three are wired through `handlers.HealthHandler` / `handlers.VersionHandler`
and registered in `internal/transport/http/router.go`; `cmd/api/main.go`
only constructs the handlers and passes them into `RouterConfig`.

## Deviations from a literal read of the spec, and why

- **Redis failures are logged, not fatal.** The spec asks for "the same
  timeout pattern" as Postgres (a bounded `context.WithTimeout` + ping),
  which this does — but a failed Redis ping only logs a `Warn` and
  continues, rather than returning an error that aborts startup. Redis
  here is cache-aside caching plus refresh-token revocation, both of
  which degrade gracefully without it (see the root `README.md`). Making
  it fatal would mean a Redis blip takes the whole API down for a feature
  that's explicitly optional infrastructure. If you'd rather it fail
  fast like Postgres, that's a one-line change (return the wrapped error
  instead of logging it).
- **Config is loaded before the logger is constructed**, matching the
  spec's literal ordering. Some reference bootstraps (e.g. Ardan Labs'
  service template) build the logger first so config-loading errors are
  themselves logged structurally. Here, `config.Load()` already returns a
  fully-formed, wrapped error, and `main()`'s stderr fallback is
  sufficient for the one failure mode that can happen before the logger
  exists — so no functionality is lost, just the log formatting of that
  one specific error path.
- **`signal.NotifyContext` instead of a hand-rolled `signal.Notify` +
  channel + `select`.** Functionally identical to "block on os/signal for
  SIGINT/SIGTERM," just the modern stdlib idiom (Go 1.16+): it gives you a
  `context.Context` that's canceled on signal, which composes directly
  with the `context.WithTimeout` calls used for the DB/Redis connects
  and lets one root context represent "the process should still be
  running" throughout `run()`.
- **`/version` required a small router change.** Adding the endpoint
  meant adding a `VersionHandler` field to `RouterConfig` and one
  `r.Get("/version", ...)` line in `router.go`, plus a new
  `internal/transport/http/handlers/version_handler.go` (the kind of
  "small supporting file" the task called out as in-scope, alongside the
  shutdown helper). No other file changed shape to support this.
