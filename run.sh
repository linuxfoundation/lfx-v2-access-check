#!/usr/bin/env sh
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

# Build and run the application.

set -e

go build -o bin/lfx-access-check ./cmd/lfx-access-check

export NATS_URL="nats://nats.lfx.svc.cluster.local:4222"
export JWKS_URL="http://heimdall.lfx.svc.cluster.local:4457/.well-known/jwks"

./bin/lfx-access-check
