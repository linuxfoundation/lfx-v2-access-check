<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Access Check Contract

This is the authoritative reference for the HTTP API exposed by
`lfx-v2-access-check`. The service is a thin HTTP wrapper around
`lfx-v2-fga-sync`'s access-check request/reply subjects.

## Endpoint

### `POST /access-check`

```http
POST /access-check?v=1
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "requests": [
    "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor",
    "committee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer"
  ]
}
```

The API version is passed as the `?v=1` query parameter, **not** in the
request body. Every entry in `requests` is a `object#relation` token. The
service appends the authenticated `@user:{principal}` suffix from the validated
Heimdall JWT before forwarding to fga-sync. Relationship-token semantics are
owned by fga-sync; this service does not define the OpenFGA model.

## Response

```json
{
  "results": [
    "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue",
    "committee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|alice\tfalse"
  ]
}
```

### Unordered-response caveat

**`results` order is not guaranteed.** fga-sync returns cached hits first and
fresh OpenFGA checks afterward, so callers must match on the full
`object#relation@user` prefix of each line, not by index. This is inherited
from the underlying NATS contract documented in
`lfx-v2-fga-sync/docs/fga-sync-contract.md`.

Lines are tab-delimited: `{object#relation@user}\t{true|false}`.

### `GET /my-grants`

```http
GET /my-grants?v=1&object_type=project
Authorization: Bearer <JWT_TOKEN>
```

Returns direct OpenFGA tuples for the authenticated caller and requested object
type. This is backed by `lfx.access_check.read_tuples`; inherited access through
parent resources is not expanded.

```json
{
  "grants": [
    "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#writer@user:auth0|alice"
  ]
}
```

## Error Mapping

| HTTP status | Cause |
| --- | --- |
| 400 Bad Request | Goa request validation failure: malformed JSON, missing required `Authorization` header, missing/unsupported `v`, empty `requests`, or invalid/missing `object_type` for `/my-grants` |
| 401 Unauthorized | JWT is present but expired, invalid, or fails JWKS validation |
| 500 Internal Server Error | Unexpected local or upstream response shape: auth claim type mismatch, JSON marshal/unmarshal failure, or malformed access-check reply |
| 503 Service Unavailable | NATS request/reply failure or timeout, read-tuples backend error, or readiness dependency failure |

The service emits only the four statuses above (per `design/access-svc.go`). There is no 403 path in this service. The Helm RuleSet authenticates callers with Heimdall and uses `allow_all`; permission decisions are returned as `true`/`false` or direct-grant data, not as gateway authorization failures.

The service never returns a partial-success body; either every request was
evaluated (200) or the call fails. Individual `false` results are normal
denial responses, not errors.

## Timeout Semantics

The service issues a single NATS request to either `lfx.access_check.request`
or `lfx.access_check.read_tuples` with a bounded timeout (default 15 seconds,
`DefaultNATSTimeout` in `pkg/constants/messaging.go`). On timeout the HTTP
response is 503 Service Unavailable with a log line, not a partial reply.

Callers should set their own client-side timeout above the service's request
timeout to allow the error path to propagate.

## Health Checks

- `GET /livez`: liveness probe; returns 200 if the service process is up.
- `GET /readyz`: readiness probe; returns 200 only when both NATS and the
  Heimdall JWKS endpoint are reachable.

## OpenAPI Spec

Available at `/_access-check/openapi.json`, `openapi.yaml`, `openapi3.json`,
`openapi3.yaml`.

## Upstream Contract

The underlying NATS request/reply contract (envelope, batch semantics,
direct-tuple reads, caching, model boundaries) is owned by `lfx-v2-fga-sync`.
Read:

`lfx-v2-fga-sync/docs/fga-sync-contract.md`
