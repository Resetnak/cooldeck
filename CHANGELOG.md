# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/) once tagged
releases begin.

## [0.2.1] - 2026-08-05

### Fixed

- `space` now marks an application for the fleet tail. The binding was declared as a literal `" "`,
  which Bubble Tea v2 never matches, so the whole fleet tail was unreachable from the keyboard
- Log lines no longer print their severity twice (`INFO  INFO  GET /healthz`) when the source
  already begins with the level
- The fleet tail footer shows its own keys (pause, follow, wrap, search, copy) instead of the
  applications list's deploy and restart hints
- Linux package install commands pointed at `/releases/latest/download/cooldeck_0.1.2_…`, a URL that
  404s after any release; they now resolve the current tag first

### Added

- `assets/tail.gif` and a dedicated fleet tail section in both READMEs, rendered from `tail.tape`
- `assets/social-preview.png`, rendered from `social.tape`
- The security section now states what CoolDeck does not protect you from: a `plaintext` token
  source, `insecure_skip_verify`, and MCP mutations running without a confirmation

## [0.2.0] - 2026-08-05

### Added

- Fleet tail: mark applications with `space`, press `t`, and read their runtime logs interleaved in
  one buffer, each line named and coloured by the application it came from

## [0.1.2] - 2026-08-05

Packaging only - the binary is identical to 0.1.0.

### Added

- Linux packages: `.deb`, `.rpm` and `.apk` for amd64 and arm64 on every release
- `install.sh`: one-line install for macOS and Linux that verifies the checksum before installing

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
- Deployments section: recent history with active-only filter, in-flight deployments first
- Deployment progress against the median of an application's last five successful builds
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

[0.2.1]: https://github.com/Resetnak/cooldeck/releases/tag/v0.2.1
[0.2.0]: https://github.com/Resetnak/cooldeck/releases/tag/v0.2.0
[0.1.2]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.2
[0.1.1]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.1
[0.1.0]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.0
