# 0001. Use Go and Charm Bubble Tea v2

## Status

Accepted

## Context

CoolDeck is a cross-platform terminal UI that must feel as polished as lazygit
or k9s, ship a single static binary, and stay maintainable as a small open-source
project.

## Decision

- Language: **Go** (version pinned in `go.mod`, currently 1.26.6+).
- TUI stack: **Charm v2** modules (`charm.land/bubbletea/v2`, `bubbles/v2`,
  `lipgloss/v2`), not the legacy `github.com/charmbracelet/...` v1 import path
  for Bubble Tea.
- CLI: **Cobra**.
- Config: **TOML** via BurntSushi/toml.

## Consequences

- Excellent static builds and GoReleaser support.
- Idiomatic Elm-architecture TUI (Model / Update / View).
- Contributors need familiarity with Bubble Tea patterns.
- Must track Charm v2 API changes separately from v1 documentation online.
