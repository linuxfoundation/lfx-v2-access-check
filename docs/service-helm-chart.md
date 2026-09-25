<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Service Helm Chart

This document owns access-check-specific chart behavior. Shared chart
conventions live in `lfx-v2-helm`; deployed values and chart pins live in
`lfx-v2-argocd`.

## Chart Shape

- **Chart path**: `charts/lfx-v2-access-check/`.
- **Service type**: thin HTTP Goa wrapper around fga-sync request/reply access
  checks. It owns no resource data, no NATS KV bucket, no tuple publishing, and
  no ExternalSecret.
- **Container env**: `PORT`, `HOST`, `DEBUG`, `AUDIENCE`, `ISSUER`, `NATS_URL`,
  and `JWKS_URL`, plus optional `app.extraEnv` and OpenTelemetry env rendered
  from `app.otel`.
- **NATS dependency**: fga-sync must be running responders for
  `lfx.access_check.request` and `lfx.access_check.read_tuples`.
- **Health probes**: liveness uses `/livez`; readiness and startup use
  `/readyz`, which checks NATS and the Heimdall JWKS endpoint.

## Routing

- **HTTPRoute paths**: exact `/access-check`, prefix `/access-check/`, prefix
  `/_access-check/`, and exact `/my-grants`.
- **RuleSet**:
  - `POST /access-check`: `oidc`, `allow_all`, `create_jwt`.
  - `GET /my-grants`: `oidc`, `allow_all`, `create_jwt`.
  - `GET|HEAD|OPTIONS /_access-check/*`: `oidc` or anonymous, `allow_all`,
    `create_jwt`.

Do not add `openfga_check` authorizer rules here. This service's job is to
return access decisions from fga-sync; it is not protecting a resource of its
own in Heimdall.

## Local Values

`charts/lfx-v2-access-check/values.local.example.yaml` only overrides the image
repository, tag, and pull policy for local chart rendering or install. The
gitignored `values.local.yaml` can add developer-only overrides without
changing the committed chart interface.
