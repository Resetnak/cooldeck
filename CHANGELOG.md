# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/) once tagged
releases begin.

## [0.1.1] - 2026-08-05

Packaging only - the binary is identical to 0.1.0.

### Added

- Homebrew cask: `brew install resetnak/tap/cooldeck`, published automatically on every tag

### Changed

- Install instructions lead with Homebrew rather than cloning the repository

## [0.1.0] - 2026-08-05

### Added

- Applications dashboard with filter, sort, responsive layout, and preview pane
- Application detail: overview, deployments, runtime logs, configuration
- Runtime and deployment logs: follow, pause, wrap, search, copy, clear, +/- lines
- Confirmed deploy / force deploy / restart / start / stop
- Command palette with fuzzy filter and disabled reasons
- Help overlay with full key map
- Instances section: mid-session switch, test connection, **add / edit / delete** local config
- Deployments section: recent history with active-only filter
- Diagnostics section with copy and file export
- Interactive `cooldeck setup` and `cooldeck theme` wizards
- Theme set: auto, dark, light, Dracula, Catppuccin, Nord, Gruvbox, Tokyo Night
- MCP server (`cooldeck mcp`): six read-only tools over stdio, mutations behind `--allow-mutations`
- Toast notifications when a deployment settles: failures always, successes for deploys this session started
- Demo mode (`--demo`) with deterministic fixtures
- OS keyring credential helpers (`cooldeck auth …`)
- Golden snapshot tests (`make test-update-golden`)
- Multi-OS CI, coverage, CodeQL, govulncheck, cross-build matrix, GoReleaser
- Documentation in English and Czech (`README.md`, `README.cs.md`, `docs/*`)
- ADRs under `docs/decisions/`

### Security

- Log sanitisation for ANSI and control characters
- Tokens excluded from logs, toasts, and diagnostics
- Browser open restricted to http(s) URLs

[0.1.1]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.1
[0.1.0]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.0
