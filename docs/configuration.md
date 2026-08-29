# Configuration

CoolDeck reads a single TOML file. Paths:

```bash
cooldeck config path
```

| OS | Default directory |
|----|-------------------|
| Linux | `$XDG_CONFIG_HOME/cooldeck` or `~/.config/cooldeck` |
| macOS | `~/Library/Application Support/cooldeck` |
| Windows | `%AppData%\cooldeck` |

Overrides:

| Variable | Effect |
|----------|--------|
| `COOLDECK_CONFIG_DIR` | Config directory |
| `COOLDECK_STATE_DIR` | Logs / diagnostics exports |
| `COOLDECK_TOKEN` | One-shot token (overrides all sources) |

## Schema

```toml
version = 1                          # required format version
default_instance = "production"      # key under [instances.*]
refresh_interval = "10s"             # dashboard poll (≥ 3s)
log_refresh_interval = "2s"          # runtime log poll
log_lines = 300                      # default log window (10–10000)
theme = "auto"                       # auto|dark|light|dracula|catppuccin|nord|gruvbox|tokyo-night
confirm_destructive_actions = true
confirm_deploy = false               # when true, normal deploy also confirms

[ui]
show_header = true
show_footer = true
nerd_font = "auto"                   # auto|on|off
compact_mode = "auto"                # auto|on|off
mouse = true

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"             # keyring|command|env|plaintext
token_key = "production"             # keyring account; defaults to instance id
# token_command = ["op", "read", "op://…"]
# token_env = "COOLIFY_TOKEN"
# token = "…"                        # only with token_source = "plaintext"
# insecure_skip_verify = false       # TLS skip - trusted networks only
# refresh_interval = "15s"           # optional per-instance override
# ssh_host = "root@coolify.example.com"  # enables the container terminal (T)
```

## Token sources

| Source | Behaviour |
|--------|-----------|
| `keyring` | OS keychain via `cooldeck auth add` (recommended) |
| `command` | External command prints token to stdout (timeout 30s) |
| `env` | Named environment variable |
| `plaintext` | Token in config file (0600); last resort |

## Validation

```bash
cooldeck config validate
```

Unknown keys are reported as warnings but do not block startup. Schema version
newer than the binary is rejected.

## State files

| Path | Content |
|------|---------|
| `$STATE_DIR/cooldeck.log` | Structured application log (redacted) |
| `$STATE_DIR/diagnostics-*.txt` | Manual diagnostics exports from the TUI |

## Themes

CLI:

```bash
cooldeck --theme nord
cooldeck theme          # interactive picker
```

In TUI: `Ctrl+T` cycles themes. Named themes ignore light/dark auto detection
for colours but still record the mode for diagnostics.
