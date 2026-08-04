# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

CoolDeck — a keyboard-first terminal dashboard (Bubble Tea / Charm v2) for [Coolify](https://coolify.io).
Single Go module, `github.com/resetnak/cooldeck`, Go 1.26.5+, no CGO.

## Commands

```bash
make check               # fmt-check + vet + test + build — run before every PR
make build               # -> bin/cooldeck, injects version/commit/date ldflags
make run                 # go run ./cmd/cooldeck --demo
make test-race
make test-update-golden  # UPDATE_GOLDEN=1 go test ./... — refresh TUI snapshots
make lint                # staticcheck ./...
make bench               # benchmarks in internal/tui/views
make vuln                # govulncheck
```

Single test / single case:

```bash
go test ./internal/tui/ -run TestGoldenViews/dashboard-wide -v
go test ./internal/coolify/ -run TestSplitDeployments
go test ./internal/tui/ -run TestGoldenViews -update   # -update flag == UPDATE_GOLDEN=1
```

CI additionally runs `golangci-lint` (config `.golangci.yml`, v2 schema, curated linter set) on Linux
and the full test matrix on ubuntu/macos/windows. `-race` is skipped on Windows.

## Architecture

Layered so the same use cases can back the TUI, a CLI subcommand, or a future MCP adapter:

```
cmd/cooldeck → internal/cli (cobra, flags, config load, service construction)
             → internal/tui (Bubble Tea model + views; presentation only)
             → internal/app (Service interface = the use cases)
               ├── internal/app/demo    deterministic fake Service
               └── internal/coolify     HTTP client + DTO → domain mapping
             → internal/domain, config, credentials, logging, platform, version
```

`docs/architecture.md` has the full picture; `docs/decisions/` holds five ADRs (Go+Charm v2,
direct REST instead of MCP, domain/DTO split, credential storage, responsive layout).
`CLAUDE_CODE_COOLIFY_TUI_SPEC.md` is the original full product spec.

### Key invariants

- **`internal/app.Service` is the seam.** Anything new that touches Coolify goes behind this
  interface, never straight from `tui` to HTTP. The demo service implements the same interface,
  which is what makes offline demo + golden tests possible.
- **Coolify DTOs never leave `internal/coolify`.** Mapping to `internal/domain` types happens in
  `mapping.go`; the rest of the app only knows domain models.
- **Errors are `*domain.Error` with an `ErrorKind`.** Callers branch on the kind, never on message
  text. `classifyHTTPError` / `classifyTransportError` in `coolify/client.go` produce them.
- **Capabilities are optimistic then downgraded.** Coolify has no permission-introspection endpoint,
  so `app.FullCapabilities()` is the starting point and `Service.downgradeForError` flips a
  capability off on 403. Features are shown disabled with a reason, never hidden.
- **Keybindings live only in `internal/tui/keys.go`.** Nothing compares key strings elsewhere; one
  `KeyMap` change updates behaviour, footer hints and the help overlay together.
- **No I/O in `View()`** or any render helper. All work happens in `tea.Cmd` goroutines.
- **Tokens never reach UI, logs, toasts or diagnostics.** `internal/logging/redact.go` enforces the
  log side; diagnostics export is secret-free by construction.

### Async lifecycle (internal/tui)

Each request kind (dashboard, detail, runtime logs, deployment logs, connect, operation) has both a
**sequence number** and a stored `context.CancelFunc` on the `Model`. A new request bumps the seq and
cancels the previous one; replies whose seq is stale are dropped. Mutating operations are serialised
via `operationInFlight`. Failed refreshes keep the last good snapshot on screen plus a stale banner —
they never wipe the list. Instance switch clears app/detail/deployment caches so fleets cannot mix.

`requestTimeout` (20s) bounds every API call; `frameInterval` (500ms) drives spinner and relative times.

## Golden tests

`internal/tui/golden_test.go` renders the demo service into `internal/tui/testdata/*.golden`.
Snapshots must stay host-independent — `normalizeGolden` strips ANSI, CRLF, spinner frames, absolute
timestamps, GOOS/GOARCH and the Go version. `.gitattributes` pins `*.golden` and `*.go` to LF.

If you change TUI chrome, layout, or copy: run `make test-update-golden` and **review the diff** —
an unexpected golden change usually means a real regression, not a stale fixture.

The demo service is seeded (`demo.DefaultSeed`) and takes a `Now func() time.Time`; keep both
injectable so tests stay deterministic.

## Config & credentials

TOML at `config.DefaultPath()` (`COOLDECK_CONFIG_DIR` overrides the dir — use it to keep tests
hermetic). `config.SchemaVersion = 1`; a file with a newer version is rejected rather than guessed at.

Token resolution order (`credentials.Resolve`): `COOLDECK_TOKEN` env override first, then the
instance's `token_source` — `keyring` (default, via `cooldeck setup` / `cooldeck auth add`),
`command`, `env`, or `plaintext` (discouraged).

## Conventions

- British English in prose and identifiers (`colour`, `behaviour`) — `misspell` is deliberately off.
  Coolify's own `CANCELLED` spelling is preserved in status parsing.
- Comments explain *why*, not *what*; that's the prevailing style — match it rather than adding
  restating comments.
- User-visible failures need an actionable title + suggestion on the `domain.Error`.
- Commit subjects: present tense, no prefix (`Add instance switch mid-session`).
