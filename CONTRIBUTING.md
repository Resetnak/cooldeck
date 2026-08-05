# Contributing

Thanks for helping improve CoolDeck.

## Development setup

```bash
git clone https://github.com/resetnak/cooldeck.git
cd cooldeck
go test ./...
make run          # demo TUI
```

Requires Go version from `go.mod`.

## Workflow

1. Prefer small, focused changes.
2. Match existing style (see nearby code; run `gofmt`).
3. Add or update tests for behaviour changes.
4. If you change TUI chrome or layout, refresh goldens when needed:

   ```bash
   make test-update-golden
   ```

5. Before opening a PR:

   ```bash
   make check
   ```

## Demo media

The GIFs and the social preview are rendered from the `*.tape` files in the repository root with
[VHS](https://github.com/charmbracelet/vhs). Each tape rebuilds `bin/cooldeck` first and runs
against `--demo` in a throwaway `COOLDECK_CONFIG_DIR`, so a recording never touches a real instance:

```bash
vhs cassette.tape   # assets/demo.gif   - one degraded app, start to finish
vhs tail.tape       # assets/tail.gif   - fleet tail across three applications
vhs outage.tape     # assets/outage.gif - stale data and recovery
vhs mcp.tape        # assets/mcp.gif    - an MCP session over stdio
vhs social.tape     # assets/social-preview.png - 1280x640 GitHub social preview
```

CI does not regenerate them; refresh the affected tape by hand when you change what it shows.

## Project layout

| Path | Role |
|------|------|
| `cmd/cooldeck` | Main |
| `internal/cli` | Cobra commands |
| `internal/tui` | Bubble Tea UI |
| `internal/app` | Use-case interface |
| `internal/coolify` | Coolify HTTP client |
| `internal/domain` | Shared models |
| `docs/` | User and architecture docs |
| `docs/decisions/` | ADRs |

## Conventions

- No I/O or network inside `View()` / pure render helpers.
- Coolify DTOs stay inside `internal/coolify`.
- Never log or render tokens.
- Keybindings live only in `internal/tui/keys.go`.
- User-visible errors should be `domain.Error` kinds with actionable titles.

## Commit messages

Use clear, present-tense summaries, for example:

```text
Add instance switch mid-session
Fix runtime log pause cancelling the wrong request
```

## Pull requests

- Describe **what** and **why**.
- Note any Coolify version assumptions for API changes.
- Do not commit secrets or real instance URLs with tokens.

## Reporting bugs

Include:

- `cooldeck version`
- OS / terminal
- Steps to reproduce
- Diagnostics export from the TUI (`4` → `e`) when relevant

See [SECURITY.md](SECURITY.md) for vulnerability reports.
