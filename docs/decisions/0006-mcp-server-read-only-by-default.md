# 0006. Ship an MCP server, read-only by default

## Status

Accepted. Extends [0002](0002-direct-coolify-api-not-mcp.md), which kept MCP out
of v1 as a transport but reserved it as a future agent interface.

## Context

Every mutating action in the TUI goes through a confirmation dialog. An agent
driving the same use cases has no one to confirm with, and the operator who
starts the server is usually not watching while it runs.

Coolify exposes no permission-introspection endpoint, so the token's real scope
is unknown until a call is refused. A server that exposes deploy by default
would therefore be as destructive as its token allows, discovered only after the
fact.

## Decision

- `internal/mcpserver` implements tools against `app.Service` only, per
  `docs/architecture.md`: no HTTP client of its own, no TUI imports.
- `cooldeck mcp` serves stdio and registers the read-only tools.
- Mutating tools (`deploy`, `restart`, `start`, `stop`) are registered only
  under `--allow-mutations`. Absent is a stronger guarantee than refused: a tool
  the agent cannot see is one it cannot decide to call.
- Domain errors are flattened to title, message and suggestion. A bare status
  code invites a retry loop; the suggestion tells the agent to stop and report.

## Consequences

- The safe default is also the quiet one: an agent asked to fix production can
  only diagnose it, and has to come back with a recommendation.
- Granting mutations is a single flag, so it is auditable in whatever launches
  the server rather than buried in config.
- `--demo` serves the same tools offline, which is how the tools are tested and
  how `mcp.tape` records the README demo.
