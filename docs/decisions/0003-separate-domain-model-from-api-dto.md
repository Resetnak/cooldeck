# 0003. Separate domain model from API DTOs

## Status

Accepted

## Context

Coolify JSON shapes change between versions and often include fields that are
noisy, sensitive, or poorly named for UI use.

## Decision

- `internal/coolify` owns private DTOs and mapping functions.
- `internal/domain` owns normalised models (`Application`, `Deployment`,
  `LogLine`, `Error`, filters, status enums).
- Presentation and future adapters import **domain** and **app**, never raw
  Coolify JSON structs.

## Consequences

- Mapping cost on every fetch (acceptable; keeps UI stable).
- API changes are isolated to one package.
- Tests can build domain fixtures without inventing Coolify payloads.
