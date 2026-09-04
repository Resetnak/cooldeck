# MCP server

`cooldeck mcp` serves your Coolify instance to an agent over the
[Model Context Protocol](https://modelcontextprotocol.io), on stdin/stdout. It is the same binary,
the same config file and the same token the TUI uses - there is no second service to install and
nothing listening on a port.

**It is read-only until you say otherwise.** An agent can see what is running and read logs; it
cannot deploy, restart, start or stop anything unless you launch the server with
`--allow-mutations`.

---

## Try it first, offline

```bash
cooldeck mcp --demo
```

This serves the deterministic demo fleet - no Coolify instance, no network, no token. The process
waits on stdin and speaks JSON-RPC, so a bare terminal will look like it hangs: that is correct, and
`Ctrl+C` exits. To see it answer, point a client at it (below) or send it a request by hand:

```bash
{ printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"probe","version":"1"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'; sleep 1; } \
  | cooldeck mcp --demo | tail -1 | jq -r '.result.tools[].name'
```

---

## Connect a client

### Claude Code

```bash
claude mcp add cooldeck -- cooldeck mcp
```

Add `--scope user` to make it available in every project rather than only the current one, and see
`claude mcp --help` for the rest. To let the agent operate as well as observe, append the flag -
everything after `--` is the command Claude Code runs:

```bash
claude mcp add cooldeck -- cooldeck mcp --allow-mutations
```

### Claude Desktop, or any client that takes a JSON config

```jsonc
{
  "mcpServers": {
    "cooldeck": {
      "command": "cooldeck",
      "args": ["mcp"]
    }
  }
}
```

`command` must be resolvable by the client, which does not necessarily inherit your shell's `PATH`.
If the server fails to start, use an absolute path - `which cooldeck` will tell you it.

### Several instances at once

The server talks to exactly one instance, chosen the same way the TUI chooses: `--instance`, or the
default from your config. Register one entry per fleet and the agent can reason across both:

```jsonc
{
  "mcpServers": {
    "cooldeck-prod":    { "command": "cooldeck", "args": ["mcp", "--instance", "production"] },
    "cooldeck-staging": { "command": "cooldeck", "args": ["mcp", "--instance", "staging", "--allow-mutations"] }
  }
}
```

That pairing is worth copying: production observed, staging operated.

---

## What the agent can do

### Always available

| Tool | Arguments | Returns |
| :--- | :--- | :--- |
| `list_applications` | - | Every application with status, project, environment and last deployment, plus the deployments in flight. **Start here** - the other tools take the UUIDs it returns. |
| `get_application` | `application_uuid` | Full detail for one application: configuration, domains, health check, recent deployments. |
| `list_deployments` | `application_uuid`, `limit` *(optional, default 20, max 200)* | Deployment history, newest first, with status, commit and duration. |
| `get_runtime_logs` | `application_uuid`, `lines` *(optional, default 100, max 2000)* | What the running container is printing now. |
| `get_deployment_logs` | `deployment_uuid` | The build log - where a failed deployment explains itself. |
| `get_instance_info` | - | URL, version, team, measured latency and which operations the token is allowed to perform. Never includes the token. |

### Only with `--allow-mutations`

| Tool | Arguments | Effect |
| :--- | :--- | :--- |
| `deploy_application` | `application_uuid`, `force` *(optional)* | Queues a deployment. Returns the deployment UUID, which the agent can follow with `list_deployments` or `get_deployment_logs`. `force` rebuilds without the Docker layer cache. |
| `restart_application` | `application_uuid` | Restarts a running application. |
| `start_application` | `application_uuid` | Starts a stopped application. |
| `stop_application` | `application_uuid` | Stops a running application. It stays stopped until started again. |

The counts are optional: leave them out and you get the default, ask for more than the ceiling and
you get the ceiling. The limits exist because every line comes back into the agent's context window,
and one careless request should not spend it all.

---

## Before you grant mutations

**There is no confirmation prompt.** The TUI asks before every destructive action; an agent has no
one to ask, so the decision moves to the moment you add the flag. Some things worth deciding
deliberately:

- **Scope the token, not just the flag.** The agent can do whatever the token can. Coolify has no
  permission-introspection endpoint, so CoolDeck assumes full capabilities and only learns otherwise
  when a call is refused with a `403`. A token limited to the instance and scopes you are willing to
  automate is a real boundary; the flag alone is a smaller one.
- **Prefer separate entries per instance.** Registering production read-only and staging with
  mutations, as above, means a mistaken instruction cannot reach production at all.
- **Nothing is hidden from you.** Every call is an ordinary Coolify API call and shows up in
  Coolify's own deployment history, attributed to the token that made it.
- **Credentials do not reach the agent.** Every frame the server writes is passed through the same
  redactor as the log file, so a token or password sitting in a build log or a commit message is
  replaced with `[REDACTED]` before it lands in a model provider's context window. Scoping the
  token is still the real boundary; the redactor is a pattern match, not a guarantee.

The reasoning behind the default is recorded in
[ADR 0006](decisions/0006-mcp-server-read-only-by-default.md).

---

## Try the tools without an agent

Anything that speaks MCP works. The MCP Inspector is convenient for poking at the surface by hand:

```bash
npx @modelcontextprotocol/inspector cooldeck mcp --demo
```

Swap `--demo` for your real flags once you trust it.

---

## Troubleshooting

| Symptom | Cause and fix |
| :--- | :--- |
| Client reports the server failed to start | `command` is not on the client's `PATH`. Use the absolute path from `which cooldeck`. |
| `no configuration file found` | The server loads the same config as the TUI. Run `cooldeck setup` first, or pass `--config`, or try `--demo`. |
| Only six tools are listed | That is the read-only default. Add `--allow-mutations` to the launch command and restart the client - the tool list is sent once, at connection. |
| A tool answers "Permission denied" | The token lacks that scope. The error carries Coolify's own suggestion; fix it in Coolify's *profile → API tokens* rather than in CoolDeck. |
| A tool answers "Request timed out" | Every call is bounded at 20 seconds, the same as in the TUI. A Coolify instance that slow will show the same symptom in the dashboard. |
| Responses look truncated | The counts are capped (see the table above). Ask for a narrower window - a specific deployment's log rather than 2000 runtime lines. |
| Garbled output or a client that cannot parse anything | Something wrote to stdout, which is the protocol transport. CoolDeck writes diagnostics to stderr by design; a wrapper script that echoes to stdout will corrupt the session. |

For anything else, `cooldeck mcp --help`, or the
[troubleshooting guide](troubleshooting.md) for problems that are really about the Coolify
connection rather than MCP.

---

## How it fits

The server lives in `internal/mcpserver` and implements tools against `app.Service` only - the same
use cases the TUI drives, with no HTTP client of its own and no imports from the TUI. Adding a tool
means adding a `Service` method, never an API call in the adapter. See
[architecture.md](architecture.md).
