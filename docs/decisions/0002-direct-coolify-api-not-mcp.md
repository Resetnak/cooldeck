# 0002. Direct Coolify REST API, not MCP as transport

## Status

Accepted

## Context

Coolify exposes a REST API. MCP is useful as a future agent interface but is
the wrong transport for an interactive operator TUI (latency, auth model,
feature coverage).

## Decision

- The TUI and CLI talk to Coolify **only** via HTTP REST (`/api/v1`).
- MCP is **out of scope for v1**.
- Architecture keeps use cases behind `app.Service` so an MCP adapter can be
  added later without rewriting Coolify access.

## Consequences

- One credential model (API tokens) shared with the Coolify web UI.
- Endpoint drift must be tracked against Coolify versions.
- No dependency on an MCP server process for normal use.
