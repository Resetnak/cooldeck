# Troubleshooting

## Cannot reach Coolify

Symptoms: offline banner, “Cannot reach Coolify”, network error toast.

1. Confirm the instance URL in config (`https://…`, no trailing path needed).
2. From the same machine: `curl -sS -o /dev/null -w '%{http_code}\n' https://your-host/api/v1/health`
3. Check TLS: self-signed hosts need `insecure_skip_verify = true` **only** on trusted networks.
4. In the TUI open **Instances** (`3`) and press `T` to re-test the active connection.
5. Use **Diagnostics** (`4`) for latency, URL, and recent errors (no secrets).

## Unauthorized / forbidden

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| 401 / unauthorized | Bad token or wrong keyring entry | `cooldeck auth add <id>` |
| 403 on deploy | Token lacks write permission | New token with deploy rights, or accept read-only |
| Actions greyed in palette | Capability downgraded after 403 | Check token scopes in Coolify |

```bash
cooldeck auth status production
echo "token set?" # never cat the secret into chat logs
```

## No applications

- Token may belong to another team.
- Coolify project empty — create an app in the web UI.
- Filter still active — press `Esc` on the list or clear with the empty-state hint.

## Logs empty or stuck

- Pause is on (`space` to resume).
- Follow off and scrolled up — `f` or `G` to jump to tail.
- Increase window with `+` (session only; does not rewrite config).
- Token may lack log read permission.

## Config not found

```bash
cooldeck setup
# or
cooldeck --demo
```

`cooldeck config path` prints where the binary looks.

## Keyring errors

- Headless Linux may need a session keyring / secret service.
- Fallback: `token_source = "env"` with a named variable, or one-shot `COOLDECK_TOKEN`.
- Avoid plaintext in config unless you accept the risk.

## Terminal too small / broken layout

- Minimum usable size is about **60×18**.
- Try `Ctrl+W` compact mode or a larger terminal.
- ASCII glyph mode activates when the locale is not UTF-8; force via code path is automatic.

## Slow over SSH

- Raise `refresh_interval` and `log_refresh_interval`.
- Avoid huge `log_lines`.
- Prefer a terminal with good bandwidth for full redraws.

## Demo mode

```bash
cooldeck --demo
```

`F2` toggles a simulated outage for offline UI testing. No network, no tokens.

## Export diagnostics for a bug report

1. Open Diagnostics (`4`).
2. Press `e` to write `$STATE_DIR/diagnostics-*.txt`.
3. Confirm the file contains **no** tokens (it should not).
4. Attach that file plus `cooldeck version` output.

## Reset local instance entry

In **Instances** (`3`), select a row and press `d` (confirm). This deletes only
local config and optional keyring data — not Coolify resources.
