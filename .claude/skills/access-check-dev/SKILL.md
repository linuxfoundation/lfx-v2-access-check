---
name: access-check-dev
description: Repo-local Go coding conventions and access-check implementation truth for lfx-v2-access-check. Auto-attaches on Go, Goa design, generated-code, Makefile, chart, docs, and local Claude guidance paths. Covers generated-code boundaries, logging, errors, request context, tests, formatting, plus the HTTP-over-NATS wrapper pattern, centralized subject constants, unordered response semantics, my-grants direct tuple reads, and the do-not-publish-index/access-mutation rule. Defers cross-repo flow and service-class discussion to lfx-skills:lfx-platform-architecture and tuple/access-check contract semantics to lfx-v2-fga-sync.
paths:
  - "**/*.go"
  - "go.mod"
  - "go.sum"
  - "Makefile"
  - "CLAUDE.md"
  - "README.md"
  - "design/**"
  - "gen/**"
  - "charts/**"
  - "docs/**"
  - ".claude/rules/**"
  - ".claude/skills/access-check-dev/**"
allowed-tools: Read, Glob, Grep, Edit, Write, Bash
---

# Development Conventions

Repo-local conventions for `lfx-v2-access-check`. Read this first when editing any Go, Goa design, or chart file in this repo.

This service is a thin HTTP wrapper over the access-check contract owned by `lfx-v2-fga-sync`. Tuple semantics, OpenFGA model boundaries, and the canonical access-check envelope live in `lfx-v2-fga-sync/docs/fga-sync-contract.md`. Keep the repo-local rule `.claude/rules/access-check-boundaries.md` aligned with that consumer-service boundary.

For platform composition, V2 service classes, and cross-service handoffs, use `lfx-skills:lfx-platform-architecture`. For cross-repo ownership routing, use `lfx-skills:lfx`.

## Generated code

- Do not edit files under `gen/` by hand. Change `design/access-svc.go` or `design/types.go`, then run `make apigen`.
- Keep hand-written implementation under `cmd/`, `internal/`, `pkg/`. Match existing package layout and constructor style.
- Preserve required license headers on new files.

## Logging

- Use `log/slog` or this repo's `pkg/log` wrapper. Never use `fmt.Println`, `fmt.Printf`, `log.Print*`, or `log.Println` for runtime service logging.
- Include stable structured fields when available: `request_id`, `principal`, `object_type`, `object_id`, `relation`, `operation`, and `subject` for NATS calls.
- Never log JWTs, bearer headers, raw token strings, secrets, or full request payloads that may contain user identifiers beyond `principal`.
- Honor `DEBUG` and any `LOG_LEVEL` env var the repo already reads.

## Errors

- Use this repo's existing error helpers in `pkg/constants/errors.go`. Do not introduce a parallel sentinel-error family.
- Preserve the error set this service emits at the Goa boundary: request validation to 400, JWT validation to 401, unexpected local/upstream response shape to 500, and NATS dependency or request/reply failures to 503. There is no 403 path in this service; the Helm RuleSet authenticates and then uses `allow_all` because authorization decisions are returned as response data. See the full mapping in `docs/access-check-contract.md`.
- Wrap upstream errors with `fmt.Errorf("...: %w", err)` so `errors.Is` and `errors.Unwrap` keep working.
- Never return raw upstream NATS error payloads or fga-sync replies to clients. Translate at the service or Goa boundary.
- This service never returns a partial-success body. Either every request was evaluated (200) or the call fails.

## Request context

- Middleware in `internal/middleware/` owns request context setup. Service-layer code must not read HTTP headers directly.
- Propagate `request_id` and `principal` only through repo helpers and typed context keys (no bare string keys).
- Forward context values into NATS calls only when the receiving subject's contract needs them.

## My grants

`GET /my-grants` is not a paginated HTTP list endpoint. It calls
`lfx.access_check.read_tuples` with JSON `{"user":"user:{principal}","object_type":"..."}`.
fga-sync paginates OpenFGA internally and this service returns the full direct-tuple
result set as `{"grants":[...]}`. Do not add local pagination parameters unless the
HTTP contract and fga-sync contract are changed together.

## Tests

- Depend on interfaces (`internal/domain/contracts/`) for NATS, auth, and config. Use the mocks under `internal/mocks/` and `test/integration/mocks.go`.
- Use table-driven tests for branching behavior. Co-locate `*_test.go` with the code under test.
- Run `make test` (race detection enabled) before handing off. Integration tests live in `test/integration/` and run against mocks, no external services required.

## Formatting and review hygiene

- Run `make fmt` and `make lint` on Go changes.
- Update `docs/access-check-contract.md` in the same change when the HTTP surface, error mapping, or timeout semantics change.
- Update `references/nats-messaging.md` when the NATS subject set, request envelope, or response shape changes.

## Access-check specifics

This repo is a proxy/consumer service in the sense of `lfx-skills:lfx-platform-architecture`. Three things are non-negotiable here.

### HTTP-over-NATS wrapper pattern

Every business endpoint follows the same shape:

1. Goa-validated HTTP request enters the handler.
2. Goa auth invokes `JWTAuth`, which validates the Heimdall-issued JWT through `internal/infrastructure/auth` and stores `HeimdallClaims` in context.
3. Handler builds the fga-sync request payload, appending `@user:{principal}` where the contract requires it.
4. `internal/infrastructure/messaging` issues a single `nats.Request` against an `lfx.access_check.*` subject with a bounded timeout.
5. Handler aggregates the reply and returns a single response. No partial success.

There is no JetStream KV ownership here. There is no publish/subscribe loop. The only NATS pattern used is synchronous request/reply.

### Centralized subject constants

All NATS subject strings live in `pkg/constants/messaging.go` (currently `AccessCheckSubject` and `ReadTuplesSubject`). Never hardcode subject strings inside handlers, service code, design files, or tests. Adding a new fga-sync subject means adding a constant first, then using it.

### Unordered response semantics

`fga-sync` returns cached hits before fresh OpenFGA checks, so reply lines are not in request order. Callers (including this service's response aggregation) must match results by the full `object#relation@user` prefix, never by index. Any future change that assumes positional matching is a bug. The contract is fixed in `lfx-v2-fga-sync/docs/fga-sync-contract.md` and surfaced to clients in `docs/access-check-contract.md`.

### Do not publish index or access-mutation messages

This service must never publish to `lfx.index.*` or to fga-sync's tuple-write subjects. It does not own any resource. It does not create, update, or delete tuples. Mutations and indexing belong to the resource-owning services. This is enforced by `.claude/rules/access-check-boundaries.md`.

## When to read which reference

| File | When to read |
| --- | --- |
| `references/goa-patterns.md` | Editing `design/` or wiring a new Goa method, error, or security scheme |
| `references/nats-messaging.md` | Adding or changing a NATS subject, request envelope, or timeout behavior |

## Handoff

- Cross-repo routing, peer-repo paths, "what owns X": `lfx-skills:lfx`.
- Platform composition, write/read/access-check flows, service classes: `lfx-skills:lfx-platform-architecture`.
- FGA tuple contract, access-check envelope, OpenFGA model: `lfx-v2-fga-sync/docs/fga-sync-contract.md`.
- Chart and deployed-values facts: `docs/service-helm-chart.md` for this repo's chart interface, then `lfx-v2-helm/docs/service-chart-patterns.md` for shared chart conventions, then `lfx-v2-argocd/docs/agent-guidance/service-chart-values-handoff.md` for deployed values and chart pins.
