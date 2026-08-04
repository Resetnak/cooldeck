# 0004. Credential storage strategy

## Status

Accepted

## Context

API tokens must not appear in the terminal, logs, screenshots, or git-tracked
config by default.

## Decision

Supported sources, in resolve order:

1. `COOLDECK_TOKEN` environment override (session / CI).
2. Per-instance `token_source`:
   - `keyring` (default recommendation)
   - `command` (password managers)
   - `env` (named variable)
   - `plaintext` (explicit opt-in only; file mode 0600)

Logging redacts secret-like values. Diagnostics and exports never include tokens.

## Consequences

- Headless environments may need `env` or `command` instead of keyring.
- Deleting a local instance can remove the keyring entry when source is keyring.
- Operators must still protect machine access; keyring is not a substitute for OS security.
