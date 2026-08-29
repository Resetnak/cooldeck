# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.5.0] - 2026-08-29

### Added

- Container terminal: `T` on an application opens an interactive shell inside
  its container, connected over SSH (`ssh_host` per instance in the config;
  the Coolify API has no exec endpoint). A picker lists every running
  container before connecting, mirroring the web terminal's choice for
  multi-container applications

### Changed

- Minimum Go version is now 1.26.6, which closes four standard-library
  advisories (`GO-2026-6218`, `GO-2026-6090`, `GO-2026-5972`, `GO-2026-5026`)
  that `govulncheck` flagged against 1.26.5

## [0.4.0] - 2026-08-09

### Added

- Fleet timeline: `t` in the Deployments section switches from the
  status-grouped table to a strictly chronological view of every deployment
  across the fleet, newest first, with relative day separators
- Env drift comparison: `x` on the applications list marks a baseline
  application, a second `x` on another one shows which environment variable
  keys exist only on one side and which differ in value; values are reduced
  to fingerprints at the API boundary and never reach the UI
- Fleet snapshot: `!` copies a secret-free markdown digest of the fleet -
  statuses, recent deployments, recent errors - to the clipboard, ready to
  paste at an AI assistant or a colleague during an incident

## [0.3.2] - 2026-08-07

### Fixed

- The Deployments section is no longer permanently empty: per-application
  deployment history now decodes Coolify's `{count, deployments}` wrapper, and
  the running queue survives the keyed-object shape Laravel emits when its
  collection is not in natural order
- The Deployments section now shows finished history across the fleet by
  merging bounded per-application history into the dashboard, instead of
  relying on the running-queue endpoint that only ever returns in-progress and
  queued deployments

## [0.3.1] - 2026-08-07

### Fixed

- Runtime logs for an application without a running container (mid-deploy,
  stopped or crashed) no longer fail with a generic "Request rejected"; the
  error now explains the state and points at the deployment log instead
- Restart, start and stop rejected because of the application's current state
  now say so instead of "Request rejected"
- A 404 on one stale resource (application deleted mid-session) no longer
  disables the capability for every other application until restart; only a
  403 downgrades capabilities, as documented

### Changed

- CI actions bumped: checkout v7, setup-go v7, upload-artifact v7,
  dependency-review-action v5, goreleaser-action v7.2.3

## [0.3.0] - 2026-08-06

The result of a full repository audit: correctness fixes, a hardened release
pipeline, and the removal of a column that never showed real data.

### Fixed

- A typo in `config.toml` no longer prevents the TUI, the MCP server or the
  `auth` commands from starting; unknown keys are reported by
  `cooldeck config validate` instead
- Adding or deleting an instance no longer freezes the whole UI (including
  `ctrl+c`) while the OS keyring responds - keyring and config writes moved off
  the event loop
- Pausing and resuming the fleet tail no longer multiplies the polling rate
- `log_refresh_interval = "0s"` no longer turns the log poll into a hot loop
- The application detail downgrades the deployments capability on a 403 instead
  of silently rendering "no deployments"
- Runtime logs now show the "older lines dropped" indicator against a real
  instance, not only in demo mode
- Recent deployments are ordered newest-first regardless of what order the API
  returns them in
- The dashboard refresh no longer parses the full build log of every deployment
  in the fleet on every poll
- Opening a link in the browser no longer leaves a zombie process behind
- An instance-switch failure no longer shows an empty toast, and a failed
  keyring cleanup after a delete surfaces as a warning

### Changed

- `install.sh` resolves the latest version from the release redirect instead of
  scraping HTML, and `COOLDECK_VERSION` works with or without the leading `v`
- Release binaries are built with `-trimpath`, and a release runs the test
  suite before publishing anything
- Local builds report the same version shape as release builds (no leading `v`)
- `make check` runs the same pinned linters as CI, so green locally means
  green in CI

### Removed

- The Project/Env column, the `project:`/`env:`/`server:` filters and the
  PRODUCTION badge: Coolify's application API returns only numeric ids, so
  these showed real data exclusively in demo mode. They can return the day the
  names are actually fetched

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

[Unreleased]: https://github.com/Resetnak/cooldeck/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/Resetnak/cooldeck/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Resetnak/cooldeck/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/Resetnak/cooldeck/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/Resetnak/cooldeck/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/Resetnak/cooldeck/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/Resetnak/cooldeck/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Resetnak/cooldeck/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/Resetnak/cooldeck/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/Resetnak/cooldeck/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Resetnak/cooldeck/releases/tag/v0.1.0
