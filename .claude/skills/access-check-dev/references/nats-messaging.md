<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# NATS Messaging (access-check)

Repo-local NATS specifics. Platform-level NATS/KV ownership lives in `lfx-skills:lfx-platform-architecture`. The canonical request/reply envelope and tuple semantics live in `lfx-v2-fga-sync/docs/fga-sync-contract.md`. The "do not hardcode subjects, do not assume reply order, do not publish index or mutation messages" rules are stated once in the parent `SKILL.md` and codified in `.claude/rules/access-check-boundaries.md`. This file covers only what is access-check-specific.

## Subjects this service calls

Both constants live in `pkg/constants/messaging.go`. New subjects added here must follow the same pattern.

| Constant | Subject | Pattern | Used by |
| --- | --- | --- | --- |
| `AccessCheckSubject` | `lfx.access_check.request` | request/reply | `check-access` handler |
| `ReadTuplesSubject` | `lfx.access_check.read_tuples` | request/reply | `my-grants` handler |

This service is not a publisher of anything else and is not a subscriber. It owns no JetStream KV bucket.

## Request/reply call shape

`internal/infrastructure/messaging/messaging_repository.go` wraps `nats.Conn.Request`. Every call passes:

- The subject from `pkg/constants/messaging.go`.
- A pre-serialized payload built by the service layer.
- A bounded timeout sourced from `pkg/constants/messaging.go` (`DefaultNATSTimeout`, 15s today).

On timeout or transport error the handler returns 503 with a log line. Unexpected reply shapes return 500. There is no retry loop in the handler and no partial-reply path. Caller-side timeouts must be larger than the NATS request timeout so the error response can propagate.

## Connection lifecycle

- Connect at startup with `MaxReconnects(3)`, `ReconnectWait(DefaultNATSReconnectWait)`, `DrainTimeout(DefaultNATSDrainTimeout)`.
- Async errors and `ClosedHandler` log through `slog`; exhausted reconnects log `ErrMsgNATSMaxReconnects`.
- Graceful shutdown calls `conn.Drain()`. Do not call `Close()` directly in service paths.
- `HealthCheck` (used by `readyz`) verifies `IsConnected`, refuses `IsClosed`/`IsDraining`, and uses `FlushWithContext` with `DefaultNATSTimeout` to confirm responsiveness.

## When subjects or payloads change

1. Update the constant in `pkg/constants/messaging.go`.
2. Update the call site and any handler that consumes the reply.
3. Coordinate with `lfx-v2-fga-sync` for the matching responder change. The envelope is theirs.
4. Update `docs/access-check-contract.md` if the change is visible to HTTP callers.
5. Update this file's subject table.
