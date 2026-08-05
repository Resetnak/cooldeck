<div align="center">

# 🎛️ CoolDeck

**Your Coolify fleet, one keystroke away.**

Deployments, logs, restarts and instance switching for [Coolify](https://coolify.io) -
from the terminal you already have open.

*No browser tab. No daemon. No token on screen. On purpose.*

<br>

[![CI](https://github.com/Resetnak/cooldeck/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/cooldeck/actions/workflows/ci.yml)
[![Security](https://github.com/Resetnak/cooldeck/actions/workflows/security.yml/badge.svg)](https://github.com/Resetnak/cooldeck/actions/workflows/security.yml)
[![Go](https://img.shields.io/badge/Go-1.26.5%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/resetnak/cooldeck)](https://goreportcard.com/report/github.com/resetnak/cooldeck)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/tested-Linux%20%C2%B7%20macOS%20%C2%B7%20Windows-yellow)](.github/workflows/ci.yml)
[![Coolify API](https://img.shields.io/badge/Coolify-API%20v1-8B5CF6)](docs/coolify-api.md)
[![Release](https://img.shields.io/github/v/release/Resetnak/cooldeck?color=brightgreen)](https://github.com/Resetnak/cooldeck/releases/latest)

<br>

**English** · [Čeština](README.cs.md)

[Quick Start](#-quick-start) · [Features](#-key-features) · [Fleet tail](#-fleet-tail) · [MCP](#-your-fleet-in-your-agent) · [Comparison](#-how-it-compares) · [Keyboard Shortcuts](#-keyboard-shortcuts) · [Installation](#-installation) · [Configuration](#-configuration) · [Security](#-security) · [Contributing](CONTRIBUTING.md)

</div>

<div align="center">
  <img src="assets/demo.gif" alt="CoolDeck in demo mode: filtering a degraded application, searching its runtime logs, confirming a deploy, and watching the new deployment appear in the active queue" width="900">

  <sub>One degraded app, start to finish: <code>/</code> to filter, <code>enter</code> for the detail, <code>l</code> for runtime logs, <code>d</code> to redeploy behind a confirmation, <code>2</code> <code>a</code> to watch it land in the active queue. Rendered from <a href="cassette.tape">cassette.tape</a>.</sub>
</div>

**Try the whole UI in one line - no Coolify instance, no token, no network:**

```bash
brew install resetnak/tap/cooldeck && cooldeck --demo
```

Prefer Go? `go install github.com/resetnak/cooldeck/cmd/cooldeck@latest`. Prefer a binary? Every
[release](https://github.com/Resetnak/cooldeck/releases/latest) ships Linux, macOS and Windows
archives plus `.deb`/`.rpm`/`.apk`. `--demo` runs against deterministic sample data - the same data
the golden tests render - so you can judge the product before you hand it a token.

---

## 💡 Why CoolDeck?

Checking whether a deploy went through should not cost you a browser tab, a login, and three clicks
through a dashboard. **CoolDeck** puts the same fleet - statuses, deployment history, runtime logs and
the deploy button - into a terminal window you can leave open next to your editor, and drives all of it
from the keyboard.

It talks to the Coolify REST API directly. No proxy, no agent, no daemon: one static binary that reads
your config, pulls a token out of your OS keyring, and renders.

```text
 COOLDECK  Demo   ● connected   DEMO                              12 apps  |  refreshed just now
────────────────────────────────────────────────────────────────────────────────────────────────
 RESOURCES        │ STATUS       APPLICATION       PROJECT / ENV     BRANCH    DEPLOYED │ billing-api
                  │ ▲ Degraded   billing-api       Billing / prod    main       47m ago │ ▲ Degraded
 > Applications 12│ ◐ Restarting ingest-dashboard  Ingest / preview  feat/…  1m 30s ago │
   Deployments  33│ ● Running    landing-web       Personal / prod   main           21s │  PRODUCTION
   Instances     1│ ○ Queued     vault-web         Personal / prod   main            4s │
   Diagnostics    │ ■ Stopped    billing-worker    Billing / prod    main        2h ago │ Project  Billing
                  │ ● Running    shipyard-api      Shipyard / prod   main        3m ago │ Branch   main
────────────────────────────────────────────────────────────────────────────────────────────────
  ↑↓ navigate   enter details   d deploy   r restart   s start/stop   / filter   ? more     1/12
```

---

## 🚀 Quick Start

1. **Look before you connect**: `cooldeck --demo` - the full UI on deterministic fake data, offline.
2. **Create a Coolify API token**: in Coolify, *profile → API tokens*. Give it the least privilege you can live with (`read` plus only the write scopes you actually want).
3. **Connect**: `cooldeck setup` walks you through URL, token and keyring storage.
4. **Use it**: `cooldeck`. Press `?` for the key map, `:` for the command palette, `/` to filter.

---

## ✨ Key Features

- **🖥️ The whole fleet on one screen**: status, project/environment, branch, last deploy and domain for every application, with filtering (`/`) and sorting (`S`).
- **🔎 Detail without a context switch**: overview, deployment history, runtime logs and configuration as tabs on the same screen.
- **🛰️ Fleet tail**: mark applications with `space`, press `t`, and read their runtime logs interleaved in one buffer, each line named and coloured by the application it came from - the view Coolify's web UI cannot give you.
- **📜 Real log ergonomics**: follow, pause, wrap, in-buffer search with `n`/`N`, copy the match, clear the buffer, `+`/`-` to widen or narrow the line window.
- **🚀 Operations behind a confirmation**: deploy, force deploy, restart, start/stop - every destructive action asks first, and only one mutation runs at a time.
- **🛟 Honest about failure**: a refresh that fails keeps the last good data on screen behind a stale banner instead of blanking the list. See [When Coolify blinks](#-when-coolify-blinks).
- **🔀 Several instances, one session**: switch fleets with `3` without restarting; add, edit and delete local instance entries from inside the TUI.
- **⌨️ Keyboard-first, mouse-optional**: vi-flavoured bindings borrowed from `lazygit` and `k9s`, a command palette (`:` / `Ctrl+K`) for the day you forget one, and a `?` overlay that always shows the truth - every hint is generated from a single `KeyMap`.
- **🎨 Eight themes, responsive layout**: auto, dark, light, Dracula, Catppuccin, Nord, Gruvbox, Tokyo Night; three-pane at 150+ columns, single column when the window is small.
- **🔐 Tokens you never see**: OS keyring by default, and tokens are kept out of the UI, the logs, the toasts and the diagnostics export by construction.
- **🤖 An MCP server in the same binary**: `cooldeck mcp` hands your fleet to an agent - read-only until you say otherwise. See [Your fleet, in your agent](#-your-fleet-in-your-agent).
- **🧪 Offline demo mode**: `--demo` is a full implementation of the same service interface, which is also what the golden snapshot tests render.

---

## 🛰️ Fleet Tail

**The one view Coolify's web UI cannot give you.** Mark the applications you care about with
`space`, press `t`, and their runtime logs arrive interleaved in a single buffer - every line named
and coloured by the application it came from. One incident, one screen, instead of a browser tab per
service.

<p align="center">
  <img src="assets/tail.gif" alt="Three applications marked with space in the CoolDeck fleet list, then t: their runtime logs interleaved in one buffer, each line prefixed and coloured by its application, searched with / and wrapped with w" width="900">

  <sub>Three services marked, one buffer: <code>space</code> to mark, <code>t</code> to tail, <code>/</code> to search across all of them, <code>w</code> to wrap. Rendered from <a href="tail.tape">tail.tape</a>.</sub>
</p>

`f` follows, `space` pauses, `c` copies the merged buffer, `esc` goes back. Coolify serves runtime
logs as whole snapshots rather than a stream, so CoolDeck polls one request per marked application
every 4 seconds, staggers them, and merges the replies by timestamp - up to five applications at a
time, and it says so when it drops the rest. The concurrency model is
[ADR 0007](docs/decisions/0007-fleet-tail-concurrency.md).

---

## 🤖 Your Fleet, In Your Agent

`cooldeck mcp` speaks the [Model Context Protocol](https://modelcontextprotocol.io) over stdin/stdout,
so an agent can ask what is running, why a build failed, and what the logs say - through the same use
cases the TUI uses. No second HTTP client, no separate token, no daemon.

<p align="center">
  <img src="assets/mcp.gif" alt="A real MCP session against CoolDeck in demo mode: the handshake, the six read-only tools, the fleet listed with statuses, the runtime logs of a degraded application revealing an upstream timeout, and the four mutating tools appearing only with --allow-mutations" width="900">

  <sub>A real JSON-RPC session against <code>--demo</code> - handshake, tool discovery, two calls, then the opt-in. Rendered from <a href="mcp.tape">mcp.tape</a>.</sub>
</p>

**It cannot touch your production by default.** The read-only surface is `list_applications`,
`get_application`, `list_deployments`, `get_runtime_logs`, `get_deployment_logs` and
`get_instance_info`. `--allow-mutations` adds `deploy_application`, `restart_application`,
`start_application` and `stop_application` - and nothing behind them asks for confirmation, because an
agent has no one to ask. Grant it deliberately, and prefer a token scoped to the instance you are
willing to let it operate.

```jsonc
// Point any MCP client at the binary you already have:
{ "mcpServers": { "cooldeck": { "command": "cooldeck", "args": ["mcp"] } } }
```

Try it before you wire it up: `cooldeck mcp --demo` serves the same tools against the offline demo
fleet, so you can watch an agent work without a Coolify instance in the loop.

📖 **[Full guide: docs/mcp.md](docs/mcp.md)** - client setup, every tool and its arguments, what to
decide before granting mutations, and troubleshooting.

---

## 🧭 How It Compares

CoolDeck is not a replacement for the Coolify web UI - it is the fast path for the handful of things
you do twenty times a day.

| Tool | Great at | Where CoolDeck differs |
| :--- | :--- | :--- |
| **Coolify web UI** | Everything - creating resources, editing env vars, managing servers | CoolDeck is read-and-operate only, but gets you from "is it up?" to "redeployed" in a few keystrokes, with no tab switch |
| **`curl` + `jq`** | Scripting, one-off queries | CoolDeck gives you the same API with statuses, history and logs in one live view, and refuses to let a typo trigger a production deploy without confirming |
| **k9s / lazydocker** | The container layer underneath | CoolDeck speaks Coolify's model - applications, projects, environments, deployments - not raw containers |

Everything it does is an ordinary Coolify API call, so nothing here locks you in or out of the web UI.

---

## 🛟 When Coolify Blinks

Dashboards that clear the screen the moment a request fails are worse than useless during an incident.
A failed refresh in CoolDeck keeps the last good snapshot, flags it as stale, and tells you how old it
is. When the instance comes back, the next refresh heals it - no restart, and you keep your place in
the list.

<p align="center">
  <img src="assets/outage.gif" alt="CoolDeck losing its connection: the applications list stays on screen behind an OFFLINE banner showing the age of the data, then recovers on the next refresh" width="900">

  <sub>Command palette, a simulated outage (<code>F2</code> in demo mode), and the recovery. Rendered from <a href="outage.tape">outage.tape</a>.</sub>
</p>

Every API call is bounded by a 20-second timeout, every request kind is cancellable, and stale replies
from a superseded request are dropped rather than rendered.

---

## 🎮 Keyboard Shortcuts

### 🧭 Navigation
| Shortcut | Action |
| :--- | :--- |
| `j` / `k` or `↑` / `↓` | Move selection |
| `g` / `G` | First / last item |
| `Ctrl+D` / `Ctrl+U` | Page down / up |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `Enter` / `Esc` | Open / back |
| `1` `2` `3` `4` | Applications · Deployments · Instances · Diagnostics |
| `q` / `Ctrl+C` | Back or quit / force quit |

### ⚡ Actions
| Shortcut | Action |
| :--- | :--- |
| `d` / `D` | Deploy / force deploy (confirms) |
| `r` | Restart (confirms) |
| `s` | Start or stop (confirms) |
| `l` / `L` | Runtime logs / build log |
| `b` / `o` | Open primary domain / repository in the browser |
| `c` | Copy application UUID |
| `S` | Cycle sort: status → name → last deploy |
| `R` | Manual refresh |

### 🔍 Filter, Search & Help
| Shortcut | Action |
| :--- | :--- |
| `/` | Filter applications (`status:`, `project:`, `env:`, `branch:`, free text) - or search the log buffer |
| `:` / `Ctrl+K` | Command palette; disabled commands show *why* |
| `?` | Help overlay with the complete key map |
| `Ctrl+T` / `Ctrl+W` | Cycle theme / toggle compact layout |

### 📜 Logs
| Shortcut | Action |
| :--- | :--- |
| `Space` | Pause / resume polling |
| `f` / `w` | Follow tail / wrap long lines |
| `/` · `n` · `N` | Search · next match · previous match |
| `c` | Copy the buffer, or the current match |
| `+` / `-` | More / fewer lines fetched (this session) |
| `Ctrl+L` | Clear the local buffer |

Full map: press `?` in the app, or read [docs/keybindings.md](docs/keybindings.md).

---

## 📦 Installation

**No runtime dependencies.** Every option below leaves you with a single static binary; only building
from source needs a toolchain (Go **1.26.5+**, no CGO).

### Option 1: Homebrew (macOS & Linux)

```bash
brew install resetnak/tap/cooldeck
cooldeck --demo
```

Upgrades come with `brew upgrade` like anything else.

### Option 2: Install script (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Resetnak/cooldeck/main/install.sh | sh
```

Detects your platform, **verifies the checksum**, and drops the binary in `~/.local/bin`. Override
with `COOLDECK_INSTALL_DIR`, or pin a version with `COOLDECK_VERSION=v0.2.1`. Read it first if you
would rather not pipe a script into a shell - [it is short](install.sh).

### Option 3: Linux packages

```bash
# Resolve the newest tag once, then pick your package manager:
VER=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
  https://github.com/Resetnak/cooldeck/releases/latest | sed 's|.*/v||')
BASE=https://github.com/Resetnak/cooldeck/releases/download/v$VER

# Debian / Ubuntu
curl -fsSLO "$BASE/cooldeck_${VER}_linux_amd64.deb"
sudo dpkg -i "cooldeck_${VER}_linux_amd64.deb"

# Fedora / RHEL
sudo rpm -i "$BASE/cooldeck_${VER}_linux_amd64.rpm"

# Alpine
curl -fsSLO "$BASE/cooldeck_${VER}_linux_amd64.apk"
sudo apk add --allow-untrusted "cooldeck_${VER}_linux_amd64.apk"
```

`.deb`, `.rpm` and `.apk` are built for `amd64` and `arm64` on every release - swap `amd64` for
`arm64` above if that is your machine.

### Option 4: Release binaries

Download an archive for your platform from [Releases](https://github.com/Resetnak/cooldeck/releases/latest),
unpack it, and put `cooldeck` on your `PATH`:

```bash
tar xzf cooldeck_*_Darwin_arm64.tar.gz     # or Linux_x86_64, Linux_arm64, Darwin_x86_64
sudo mv cooldeck /usr/local/bin/
cooldeck version
```

Windows ships as a `.zip`. Every release carries a `checksums.txt`; verify before you trust it:

```bash
shasum -a 256 -c checksums.txt --ignore-missing
```

Archives for Linux, macOS and Windows (`amd64` & `arm64`) are built by
[GoReleaser](.goreleaser.yaml) from `v*` tags.

### Option 5: `go install`

```bash
go install github.com/resetnak/cooldeck/cmd/cooldeck@latest
```

### Option 6: Build from source

```bash
git clone https://github.com/Resetnak/cooldeck.git
cd cooldeck
make build          # -> bin/cooldeck, with version/commit/date baked in
./bin/cooldeck --demo
install -m 0755 bin/cooldeck ~/.local/bin/cooldeck
```

---

## ⚙️ Configuration

CoolDeck reads one TOML file. Find it - and check it - with:

```bash
cooldeck config path
cooldeck config validate
```

| OS | Default directory |
| :--- | :--- |
| **Linux** | `$XDG_CONFIG_HOME/cooldeck` or `~/.config/cooldeck` |
| **macOS** | `~/Library/Application Support/cooldeck` |
| **Windows** | `%AppData%\cooldeck` |

```toml
version = 1                          # schema version; a newer one is rejected, never guessed at
default_instance = "production"
theme = "auto"                       # auto|dark|light|dracula|catppuccin|nord|gruvbox|tokyo-night
refresh_interval = "10s"             # dashboard poll (minimum 3s)
log_refresh_interval = "2s"
log_lines = 300                      # default log window (10–10000)
confirm_destructive_actions = true
confirm_deploy = false               # set true to confirm ordinary deploys too

[ui]
nerd_font = "auto"                   # auto|on|off
compact_mode = "auto"
mouse = true

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"             # keyring|command|env|plaintext
token_key = "production"
```

`COOLDECK_CONFIG_DIR` moves the whole directory - handy for keeping experiments away from your real
setup. Full reference: [docs/configuration.md](docs/configuration.md).

### 🔑 Token sources

Resolved in this order: `COOLDECK_TOKEN` in the environment always wins, otherwise the instance's
`token_source` decides.

| Source | Behaviour |
| :--- | :--- |
| **`keyring`** | OS keychain, written by `cooldeck setup` or `cooldeck auth add` - **recommended** |
| `command` | Runs an external command and reads the token from stdout (e.g. `["op", "read", "op://…"]`) |
| `env` | Reads a named environment variable |
| `plaintext` | Token in the config file (mode `0600`) - last resort |

```bash
cooldeck auth add production      # store a token in the keyring
cooldeck auth status production   # is one there? (never prints it)
COOLDECK_TOKEN=… cooldeck         # one-shot, nothing written anywhere
```

---

## 🖥️ CLI

```text
cooldeck                          the TUI dashboard
cooldeck --demo                   offline demo data, no Coolify needed
cooldeck --instance production    start on a specific instance
cooldeck --theme catppuccin       theme override for this run
cooldeck --debug                  structured debug log (redacted)

cooldeck mcp                      serve the instance to an agent over MCP (read-only)
cooldeck mcp --allow-mutations    ... and let it deploy, restart, start and stop

cooldeck setup                    first-run wizard: URL, token, keyring
cooldeck theme                    interactive theme picker with live preview
cooldeck auth add|status|delete <instance>
cooldeck config path|validate
cooldeck version                  version, commit, build date
```

---

## 🔒 Security

- **Tokens stay out of sight**: never rendered in the UI, never written to logs, toasts or the diagnostics export - [`internal/logging/redact.go`](internal/logging/redact.go) enforces the log side, and the diagnostics dump is secret-free by construction.
- **Log output is sanitised**: raw ANSI control sequences from a remote log stream cannot repaint your terminal.
- **Only `http`/`https`** URLs are ever handed to the browser.
- **Deleting an instance** removes the *local* config entry and its keyring item. It never touches anything in Coolify.
- **Permissions degrade gracefully**: Coolify has no permission-introspection endpoint, so CoolDeck assumes full capabilities and switches individual features off on a `403` - showing them disabled with a reason rather than hiding them.
- **What it does not protect you from**: the config file is written `0600` but a `plaintext` token source still puts the token on disk; `insecure_skip_verify = true` really does disable TLS verification for that instance; and an MCP client started with `--allow-mutations` can deploy, restart, start and stop without a confirmation, because an agent has nobody to ask. All three are opt-in, and all three are worth a second thought.

Reporting a vulnerability: [SECURITY.md](SECURITY.md).

---

## 🏗️ Architecture

Layered so that the same use cases back the TUI, a CLI subcommand and the MCP server:

```text
cmd/cooldeck → internal/cli        cobra, flags, config, service construction
             → internal/tui        Bubble Tea model + views (presentation only, no I/O)
             → internal/mcpserver  MCP tools over stdio (no HTTP client, no TUI imports)
             → internal/app        Service interface = the use cases
               ├── app/demo        deterministic fake service (demo mode + golden tests)
               └── coolify         HTTP client + DTO → domain mapping
             → internal/domain, config, credentials, logging, platform, version
```

Built on [Bubble Tea / Charm v2](https://github.com/charmbracelet/bubbletea). Every TUI screen is
covered by golden snapshots rendered from the demo service, so a layout regression fails CI instead of
shipping.

| Doc | |
| :--- | :--- |
| [Architecture](docs/architecture.md) | Layers, message flow, async lifecycle |
| [Decisions](docs/decisions/) | Seven ADRs: Go + Charm, direct REST over MCP, domain/DTO split, credential storage, responsive layout, MCP server, fleet-tail concurrency |
| [Coolify API](docs/coolify-api.md) | Which endpoints are used and how |
| [MCP server](docs/mcp.md) | Connecting an agent: clients, tools, safety, troubleshooting |
| [Configuration](docs/configuration.md) | Full schema reference |
| [Keybindings](docs/keybindings.md) | The complete key map |
| [Troubleshooting](docs/troubleshooting.md) | Common failures and what they mean |
| [Changelog](CHANGELOG.md) | What has shipped |

### Development

```bash
make check               # fmt-check + vet + test + build - run this before every PR
make run                 # go run ./cmd/cooldeck --demo
make test-race
make test-update-golden  # refresh the TUI snapshots, then *read the diff*
make lint                # staticcheck
make bench               # view rendering benchmarks
make vuln                # govulncheck
```

The two GIFs in this README are generated, not hand-recorded: `vhs cassette.tape` and
`vhs outage.tape` rebuild the binary and re-record against `--demo`, so they cannot drift from the
working tree. Both run in a throwaway `COOLDECK_CONFIG_DIR` under `/tmp` and touch nothing of yours.

---

## 🗺️ Roadmap

**Shipped:** applications dashboard, detail with logs, confirmed mutations, deployments queue,
deployment outcome notifications, multi-instance management, diagnostics, eight themes, demo mode,
an MCP server on the same `app.Service`, golden tests, multi-OS CI.

**Next:** richer in-TUI token sources beyond the keyring, optional read-only views for services,
databases and servers, and a non-interactive CLI for scripts and CI.

---

## 🤝 Contributing

Bug reports, feature requests and PRs are welcome - see [CONTRIBUTING.md](CONTRIBUTING.md) for the
local setup, the golden-test workflow and what `make check` expects before review. Participation is
covered by the [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

[MIT](LICENSE) © 2026 Alexandr Rešetňak
