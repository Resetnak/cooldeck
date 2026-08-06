# Coolify API notes

CoolDeck uses the Coolify HTTP API under `/api/v1` with:

```http
Authorization: Bearer <token>
Accept: application/json
```

The implementation lives in `internal/coolify`. DTO types are private to that
package; public code consumes `domain` models via `app.Service`.

## Source of truth

Endpoint paths and payloads **must** be verified against the Coolify version
you target (official docs / OpenAPI). This document describes how CoolDeck
uses the API, not a frozen vendor contract.

## Operations used

| Use case | Service method | Typical Coolify surface |
|----------|----------------|-------------------------|
| Health / version | `Connect` | `health`, `version`, `teams/current` |
| Application list | `Dashboard` | applications list (+ optional enrichment) |
| Active deployments | `Dashboard` | deployments queue / active list |
| Application detail | `ApplicationDetail` | application by UUID |
| Deployment history | `ApplicationDetail` / `Deployments` | deployments for app |
| Runtime logs | `RuntimeLogs` | application logs (bounded lines) |
| Build logs | `DeploymentLogs` | deployment logs payload |
| Deploy | `Deploy` | deploy / force deploy |
| Restart / start / stop | `Restart` / `Start` / `Stop` | lifecycle endpoints |

Exact paths are centralised in the Coolify client so they can be updated in one
place when Coolify changes.

## Error mapping

HTTP failures become `domain.Error` with a kind:

| Kind | Typical cause |
|------|----------------|
| `network` | DNS, TLS, timeout, connection refused |
| `unauthorized` | 401 - bad or missing token |
| `forbidden` | 403 - token lacks permission (capability downgrade) |
| `not_found` | 404 |
| `rate_limited` | 429 |
| `cancelled` | context cancel (superseded request; usually silent) |
| `decode` | unexpected JSON |

## Capabilities

There is no Coolify “what can this token do?” endpoint. CoolDeck:

1. Starts with full capabilities.
2. Downgrades specific flags when a mutation returns 403.
3. Surfaces disabled actions in the palette and action guards.

## Logs

Coolify often returns logs as a **single string**, not a stream. CoolDeck:

- polls snapshots on an interval
- parses lines with `domain.ParseLogPayload`
- sanitises ANSI/control characters before rendering
- keeps only the newest `log_lines` window

## Security

- Never log `Authorization` headers or token values.
- Never open non-http(s) URLs from API data (`platform.OpenURL`).
- Treat FQDNs and repo URLs as untrusted display/open inputs.

## Demo mode

`--demo` never calls Coolify. `internal/app/demo` implements the same
`app.Service` interface with seeded fixtures for UI development and golden tests.
