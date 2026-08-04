# 0005. TUI responsive layout

## Status

Accepted

## Context

Users run CoolDeck on large desktops, laptops, and small SSH panes. A single
fixed layout either wastes space or becomes unusable.

## Decision

Breakpoints in `internal/tui/theme/layout.go`:

| Tier | Rough width | Behaviour |
|------|-------------|-----------|
| Too small | &lt; 60 or height &lt; 18 | “Terminal too small” screen |
| Compact | narrow / short | Single column, minimal chrome |
| Standard | ≥ 80 | Top tabs, reduced columns |
| Wide | ≥ 120 | Sidebar + optional preview |

Layout is recomputed on resize and compact toggle, never inside `View()` from
scratch without the shared `Layout` value. Columns use priority so status/name
survive when space is tight.

## Consequences

- Consistent geometry across header/body/footer for one frame.
- Golden tests cover wide/standard/compact and offline/empty states.
- Very small terminals cannot show full tables; that is intentional.
