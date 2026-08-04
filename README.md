# CoolDeck

A keyboard-first terminal dashboard for Coolify, built with Go and Charm v2.

CoolDeck currently provides a responsive applications dashboard, filtering,
sorting, application detail, automatic refresh, dark/light themes, safe
credential loading, and a deterministic demo mode. The Coolify client already
supports application and deployment reads, logs, and lifecycle operations;
the remaining log/deployment/mutation screens are still in progress.

## Run it

Requirements: Go 1.26.5 or newer.

```bash
go build -o cooldeck ./cmd/cooldeck
./cooldeck --demo
```

Useful commands:

```bash
cooldeck version
cooldeck config path
cooldeck config validate
cooldeck auth add production
cooldeck auth status production
cooldeck auth delete production
```

## Configuration

CoolDeck uses the platform config directory (`~/.config/cooldeck` on most
Linux systems, `~/Library/Application Support/cooldeck` on macOS). Print the
exact path with `cooldeck config path`.

```toml
version = 1
default_instance = "production"
refresh_interval = "10s"
log_refresh_interval = "2s"
log_lines = 300
theme = "auto"
confirm_destructive_actions = true

[ui]
show_header = true
show_footer = true
nerd_font = "auto"
compact_mode = "auto"
mouse = true

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"
token_key = "production"
```

Tokens should live in the OS keyring. For a one-off session, use the
`COOLDECK_TOKEN` environment variable. Plaintext tokens are supported only as
an explicit config opt-in and are never recommended.

## Development

```bash
make check
make run
make test-race
```

The full product and UX specification is in
[`CLAUDE_CODE_COOLIFY_TUI_SPEC.md`](CLAUDE_CODE_COOLIFY_TUI_SPEC.md).
