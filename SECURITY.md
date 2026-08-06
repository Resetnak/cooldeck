# Security Policy

## Supported versions

Security fixes target the latest release on the default branch. There is no
long-term support branch yet.

## What CoolDeck handles

- Coolify API tokens (keyring / env / command / optional plaintext config)
- Application and deployment metadata from Coolify
- Application and build log text (treated as **untrusted**)

## Reporting a vulnerability

Please **do not** open a public issue for exploitable security bugs.

Report privately through either channel:

- [GitHub security advisory](https://github.com/Resetnak/cooldeck/security/advisories/new) (preferred)
- Email the maintainer: <mail@resetnak.cz>

Include:

- description of the issue
- impact (token leak, RCE via logs, path traversal, etc.)
- reproduction steps or PoC
- affected commit or version if known

You will get an acknowledgement within 72 hours, and a fix lands before any
public disclosure.

## Non-vulnerabilities

Please use normal issues for:

- missing Coolify features
- UI polish
- documentation gaps
- tokens stored with `token_source = "plaintext"` by user choice

## Hardening tips for operators

- Prefer `token_source = "keyring"`.
- Scope Coolify tokens to the minimum permissions required.
- Do not commit `config.toml` with plaintext tokens.
- Treat exported diagnostics as operational data (no tokens, but may include hostnames).
