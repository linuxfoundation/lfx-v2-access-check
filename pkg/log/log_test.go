// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package log

import (
	"context"
	"log/slog"
	"testing"
)

func TestAppendCtx(t *testing.T) {
	ctx := context.Background()

	// Test appending a single attribute
	ctx = AppendCtx(ctx, slog.String("request_id", "test-123"))

	// Verify context has the attribute
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); !ok || len(attrs) != 1 {
		t.Error("Expected one attribute in context")
	}

	// Test that we can append multiple attributes
	ctx = AppendCtx(ctx, slog.String("user_id", "user-456"))

	// Verify we can still get the attributes
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); !ok || len(attrs) != 2 {
		t.Error("Expected two attributes in context")
	}
}

func TestInitStructureLogConfig(_ *testing.T) {
	// Test that InitStructureLogConfig doesn't panic
	InitStructureLogConfig()

	// Test that we can log after initialization
	ctx := context.Background()
	ctx = AppendCtx(ctx, slog.String("test", "value"))

	// This should not panic
	slog.InfoContext(ctx, "test message")
}
