# Architecture

CoolDeck is layered so the same business logic can drive the TUI today and a
CLI or MCP adapter later without rewriting Coolify integration.

```text
┌─────────────────────────────────────────────────────────────┐
│  cmd/cooldeck          entrypoint                           │
│  internal/cli          cobra flags, setup, auth, theme      │
├─────────────────────────────────────────────────────────────┤
│  internal/tui          Bubble Tea model, views, chrome      │
├─────────────────────────────────────────────────────────────┤
│  internal/app          Service interface (use cases)        │
│  internal/app/demo     deterministic fake Service           │
├─────────────────────────────────────────────────────────────┤
│  internal/coolify      HTTP client + DTO → domain mapping    │
│  internal/domain       status, filter, logs, errors         │
│  internal/credentials  keyring / env / command tokens       │
│  internal/config       TOML load/save/validate              │
└─────────────────────────────────────────────────────────────┘
```

## Layers

| Package | Responsibility |
|---------|----------------|
| `cli` | Process boundaries: flags, config load, service construction, logging |
| `tui` | Pure presentation + event loop; no direct Coolify HTTP |
| `app` | Use-case interface (`Connect`, `Dashboard`, `Deploy`, …) |
| `coolify` | REST transport, DTO parsing, capability downgrade on 403 |
| `domain` | Normalised models, filters, log sanitisation, error kinds |
| `credentials` | Resolve/store/delete tokens without exposing secrets to logs |
| `config` | User preferences and instance list |

## Data flow

1. CLI loads config and resolves a token for the selected instance.
2. `coolify.New` builds an `app.Service` bound to that instance.
3. TUI `Model.Init` calls `Connect`, then `Dashboard` on a background command.
4. Results arrive as typed messages with sequence numbers.
5. Views hold selection/scroll state; the model owns async lifecycle and chrome.

```text
KeyPress → handleKey → tea.Cmd (goroutine)
                ↓
         API / clipboard / browser
                ↓
         Msg (seq checked) → mutate Model → View()
```

## Bubble Tea message flow

- **No I/O in `View()`.** Rendering only reads model/view fields.
- **Sequence numbers** per request kind (`dashboardSeq`, `runtimeLogsSeq`, …)
  drop stale replies when a newer request has started.
- **Cancellation:** each request kind keeps a `context.CancelFunc`. Starting a
  new request cancels the previous one; leaving a screen cancels its pollers.
- **Toasts** are pushed as messages and pruned on a frame tick (capped stack).

## Request lifecycle

| Kind | Trigger | Supersession |
|------|---------|--------------|
| Connect | startup, instance switch, test (`T`) | new connect cancels previous |
| Dashboard | connect success, `R`, refresh tick | background refresh skips if one in flight; manual supersedes |
| Detail | open application | new open cancels previous |
| Runtime logs | open tab + poll tick | cancel on tab leave / pause |
| Deployment logs | open build log | cancel on close |
| Operation | confirmed deploy/restart/… | one at a time (`operationInFlight`) |

## Cache and offline behaviour

- Last successful dashboard snapshot stays on screen when a refresh fails.
- A stale banner + error toast explain degradation; the list is not wiped.
- Instance switch clears application/detail/deployment caches so fleets never mix.

## Credentials

Resolution order (see `credentials.Resolve`):

1. `COOLDECK_TOKEN` environment override
2. Instance `token_source` (`keyring` | `command` | `env` | `plaintext`)

Tokens never appear in slog output, diagnostics, or toasts. Keyring entries are
removed when an instance is deleted from local config (optional cleanup).

## Capability detection

`app.Capabilities` starts optimistic (`FullCapabilities`). Coolify has no
permission-introspection endpoint, so mutating calls that return 403 downgrade
the matching capability. The palette and actions show **disabled + reason**
instead of hiding features.

## Mid-session instance switch

`tui.Options.OpenService` rebuilds an `app.Service` for another config ID
without restarting the process. The CLI supplies a factory that reloads config,
resolves credentials, and constructs `coolify.Service`.

## Future MCP adapter

An MCP server should implement tools against `app.Service` only:

- no duplicate HTTP client
- no TUI imports
- same domain errors and capability rules

The TUI remains one of several adapters over the use-case layer.

## Performance notes

- Theme/styles are built once per theme change, not per frame.
- Application filter/sort runs only on data or filter/sort change (`reproject`).
- Log lines are capped by session `logLines` (config default 300).
- Toast and recent-error buffers are bounded.
- Goroutines are only those started by Bubble Tea commands; shutdown cancels in-flight work.
