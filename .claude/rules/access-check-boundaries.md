---
description: Access-check service boundaries, do not publish to indexer/fga-sync, centralize subject constants, no order assumptions
paths:
  - '**/*.go'
  - 'design/**'
  - 'pkg/constants/**'
  - 'charts/**'
  - 'docs/**'
  - '.claude/**'
---

<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Access-check service boundaries

- Do not add resource ownership, KV storage, index publishing, or access mutation publishing in this service. It is a synchronous request/reply consumer over `lfx-v2-fga-sync`.
- Centralize NATS subject constants. Never hardcode subject strings inside handlers, service code, or design files.
- Do not assume any ordering of responses returned from fga-sync. Match results by object, relation, and user identity, not by incidental NATS response position.
- Treat `lfx-v2-fga-sync` as the authoritative source for permission semantics, tuple writes, and the check response format.
- Do not add Heimdall `openfga_check` authorizer rules for `/access-check` or `/my-grants` in this chart. The RuleSet authenticates callers; fga-sync authorization decisions are returned by this service as data.

Canonical contract: `lfx-v2-fga-sync/docs/fga-sync-contract.md`.
