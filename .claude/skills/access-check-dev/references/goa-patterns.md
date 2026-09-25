<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Goa Patterns (access-check)

Repo-local Goa specifics. Generic Goa concepts and V2 service-class context live in `lfx-skills:lfx-platform-architecture`. Generic Go conventions live in the parent `SKILL.md`. This file documents only what is specific to this repo's design surface.

## Design layout

- Designs live at the repo root in `design/`:
  - `design/access-svc.go`: API, security scheme, methods, HTTP transport.
  - `design/types.go`: shared types.
- Generated code lands in `gen/` after `make apigen`. Never hand-edit `gen/`.

## Methods exposed today

| Method | HTTP | Purpose |
| --- | --- | --- |
| `check-access` | `POST /access-check?v=1` | Bulk access check, body is `requests: []string` |
| `my-grants` | `GET /my-grants?v=1&object_type=...` | Direct grants for the caller on an object type |
| `livez` | `GET /livez` | Liveness, plaintext OK |
| `readyz` | `GET /readyz` | Readiness, plaintext OK, checks NATS and JWKS |
| `Files(...)` | `GET /_access-check/openapi*` | Self-served OpenAPI specs |

## Local deviations

- API version is a query param (`v=1`), not a body field or path segment. Enforced via `Param("version:v")` and `Enum("1")`.
- Bearer token is captured with `Token("bearer_token", ...)` and `Header("bearer_token:Authorization")`. The service validates the Heimdall-issued JWT via `JWTAuth` before reading the principal from `HeimdallClaims`.
- `requests` uses `MinLength(1)`, so the HTTP transport rejects an empty JSON array before service code can normalize it to an empty result.
- No ETag or If-Match wiring. This service has no mutable resource state.
- Error set per method is the standard four: `BadRequest`, `Unauthorized`, `InternalServerError` (Fault), `ServiceUnavailable` (Temporary + Fault). Map status codes in the `HTTP(...)` block.
- Examples in payload/result attributes reuse constants from `pkg/constants` (`ExampleProjectAction`, `ExampleCommitteeAction`) so the OpenAPI examples stay aligned with the contract.

## Contract source of truth

The HTTP contract (request, response, unordered-response caveat, error mapping, timeout, and direct-grant behavior) is owned by `docs/access-check-contract.md`. Update that file in the same change as any design-level break or addition.
