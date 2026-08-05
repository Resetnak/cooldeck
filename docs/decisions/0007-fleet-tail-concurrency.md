# 0007. One request per tailed application, bounded by a shared sequence

## Status

Accepted.

## Context

Every other request kind in `internal/tui` keeps exactly one request in flight:
a sequence number plus a stored `context.CancelFunc` on the `Model`, where a new
request cancels the previous one and stale replies are dropped by sequence.

The fleet tail breaks that shape by construction. Coolify serves logs as whole
snapshots rather than a stream, so tailing N applications means N independent
polls, repeatedly, for as long as the screen is open. A single cancel function
cannot express that, and one cancel function per application would put an
unbounded map of them on the `Model`.

Those polls also land on a rate limit that the whole session shares -
`config.MinRefreshInterval` exists for exactly that reason.

## Decision

- The tail keeps **one sequence number for the whole session**, `tailSeq`, and
  no cancel functions. Every source's replies carry it; leaving the tail bumps
  it, which orphans all of them at once.
- Each request still gets `requestTimeout`, so nothing outlives the screen by
  more than 20 seconds even though nothing cancels it.
- **At most `views.MaxTailApps` (5) sources.** The cap is a rate-limit guard
  first and a readability one second.
- Polls are **staggered** across the interval rather than fired together, and
  the tail interval is deliberately slower than the single-application one.
- A failure degrades **one source**, not the screen. The failing source is
  labelled in the header and retried with a backoff; a rate limit is obeyed for
  exactly as long as Coolify asked for.

## Consequences

- The invariant in CLAUDE.md is now "one in flight per request kind, except the
  fleet tail" rather than an unqualified rule. Stated explicitly so the next
  reader does not take the exception as licence.
- Cancellation is coarser: leaving the tail does not abort the HTTP requests
  already issued, it only ignores their replies. With a 20-second ceiling and at
  most five of them, that is a bounded cost rather than a leak.
- Raising the cap means re-examining the rate-limit arithmetic, not just the
  constant. At five sources on a four-second interval the tail averages a little
  over one request per second, which is below what the dashboard refresh already
  costs.
