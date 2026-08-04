# CoolDeck

**A fast, keyboard-first terminal dashboard for [Coolify](https://coolify.io).**

Monitor deployments, open logs, restart apps, and switch instances — without leaving the terminal.

[![CI](https://github.com/resetnak/cooldeck/actions/workflows/ci.yml/badge.svg)](https://github.com/resetnak/cooldeck/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/resetnak/cooldeck)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> 🇨🇿 **Česká verze:** [README.cs.md](README.cs.md)

```text
┌ cooldeck · production · online ────────────────────────── 12 apps · 3s ago ┐
│ Applications │ Deployments │ Instances │ Diagnostics                        │
│                                                                             │
│  STATUS      APPLICATION         PROJECT / ENV     BRANCH    DEPLOYED       │
│  ▶ Running   stimustop-api       StimuStop / prod  main      3m ago         │
│    Failed    weektraq-web        Weektraq / stage  develop   2h ago         │
│                                                                             │
└ ↑↓ move  enter detail  / filter  d deploy  : commands  ? help ─────────────┘
```

## Why CoolDeck?

| | |
|--|--|
| **Fast** | Direct Coolify REST API — no browser, no MCP hop |
| **Safe** | Tokens in the OS keyring; destructive actions always confirm |
| **Polished** | Charm v2 UI, themes, command palette, responsive layout |
| **Demoable** | `cooldeck --demo` works offline with realistic data |

## Install

**Requirements:** Go **1.26.5+**

```bash
git clone https://github.com/resetnak/cooldeck.git
cd cooldeck
make build
./bin/cooldeck --demo
```

Release binaries (Linux / macOS / Windows) ship via GoReleaser on `v*` tags.

```bash
go install github.com/resetnak/cooldeck/cmd/cooldeck@latest   # when published
```

## 60-second start

```bash
# 1) Explore without Coolify
cooldeck --demo

# 2) Real instance — interactive wizard
cooldeck setup

# 3) Daily use
cooldeck
cooldeck --instance production
cooldeck --theme catppuccin
```

**Coolify token:** create an API token in Coolify (profile → API tokens). Prefer least privilege (`read` + only the write scopes you need).

## Features

- **Applications** — status, project/env, branch, last deploy, domain; filter & sort
- **Detail** — overview, deployment history, runtime logs, configuration
- **Logs** — follow / pause / wrap / search / copy / clear / `+/-` line window
- **Actions** — deploy, force deploy, restart, start/stop (with confirmation)
- **Deployments** — recent history + active-only filter (`a`)
- **Instances** — mid-session switch, test connection, **add / edit / delete** local config
- **Diagnostics** — environment dump; copy or export (no secrets)
- **Command palette** (`:` / `Ctrl+K`) and **help** (`?`)
- **Themes** — auto, dark, light, Dracula, Catppuccin, Nord, Gruvbox, Tokyo Night

## Keybindings (essentials)

| Keys | Action |
|:-----|:-------|
| `j` `k` · `↑` `↓` | Move |
| `enter` · `esc` | Open · back |
| `/` | Filter or log search |
| `:` · `Ctrl+K` | Command palette |
| `?` | Help |
| `1`–`4` | Apps · Deploys · Instances · Diagnostics |
| `d` `D` `r` `s` | Deploy · force · restart · start/stop |
| `l` `L` | Runtime logs · build log |
| `c` | Copy UUID / logs |
| `Ctrl+T` · `Ctrl+W` | Theme · compact mode |

Full map: in-app `?` or [docs/keybindings.md](docs/keybindings.md).

## Configuration

```bash
cooldeck config path
cooldeck config validate
```

```toml
version = 1
default_instance = "production"
theme = "auto"
refresh_interval = "10s"

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"    # recommended
token_key = "production"
```

Tokens: **keyring** (default via setup/auth) · `command` · `env` · `plaintext` (discouraged).  
One-shot: `COOLDECK_TOKEN=… cooldeck`.

Details: [docs/configuration.md](docs/configuration.md)

## CLI

```text
cooldeck                 TUI dashboard
cooldeck --demo          Offline demo data
cooldeck setup           first-run wizard
cooldeck theme           theme picker
cooldeck auth add|status|delete <instance>
cooldeck config path|validate
cooldeck version
```

## Security

- Tokens never appear in the UI, logs, toasts, or diagnostics
- Log output is sanitised (no raw ANSI control sequences)
- Only `http`/`https` URLs are opened in the browser
- Deleting an instance removes **local** config (+ keyring entry), not Coolify resources

See [SECURITY.md](SECURITY.md).

## Development

```bash
make check               # fmt + vet + test + build
make test-race
make test-update-golden  # refresh UI snapshots
make run                 # --demo
make bench
```

| Doc | |
|-----|--|
| [Architecture](docs/architecture.md) | Layers & message flow |
| [Coolify API](docs/coolify-api.md) | Integration notes |
| [Troubleshooting](docs/troubleshooting.md) | Common failures |
| [Contributing](CONTRIBUTING.md) | PR workflow |
| [Changelog](CHANGELOG.md) | What shipped |
| [Spec](CLAUDE_CODE_COOLIFY_TUI_SPEC.md) | Full product spec |

## Roadmap

**Now:** dashboard, logs, mutations, instances (switch/add/edit/delete), diagnostics, goldens, CI.

**Next:** richer in-TUI token sources beyond keyring, optional read-only services/databases/servers, MCP adapter over the same `app.Service`.

## License

[MIT](LICENSE) · contributions welcome.
