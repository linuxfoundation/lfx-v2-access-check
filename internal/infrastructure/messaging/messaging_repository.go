// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package messaging provides NATS-based messaging infrastructure for the access service.
package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
)

// tracer is safe to initialize at package level. otel.Tracer() returns a
// delegating tracer that forwards to whatever TracerProvider is registered at
// call time, so otel.SetTracerProvider() updates it regardless of init order.
var tracer = otel.Tracer("github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/messaging")

// messagingReplyBodySizeKey records the size of the NATS reply payload.
// Named to distinguish it from messaging.message.body.size (the outbound request).
// There is no semconv standard attribute for reply size yet.
const messagingReplyBodySizeKey = attribute.Key("messaging.message.reply.body.size")

// natsHeaderCarrier adapts nats.Header to the OTel TextMapCarrier interface
// so trace context can be injected into and extracted from NATS message headers.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	vals := c[key]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (c natsHeaderCarrier) Set(key string, value string) {
	c[key] = []string{value}
}

func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

var _ propagation.TextMapCarrier = natsHeaderCarrier{}

type messagingRepository struct {
	conn *nats.Conn
}

// NewMessagingRepository creates a new NATS-based messaging repository
func NewMessagingRepository(natsURL string) (contracts.MessagingRepository, error) {
	slog.Info("Connecting to NATS", "nats_url", natsURL)

	// Configure NATS connection options with basic error handling from reference
	opts := []nats.Option{
		nats.MaxReconnects(3),
		nats.ReconnectWait(constants.DefaultNATSReconnectWait),
		nats.DrainTimeout(constants.DefaultNATSDrainTimeout),
		nats.ErrorHandler(func(_ *nats.Conn, s *nats.Subscription, err error) {
			if s != nil {
				slog.Error("async NATS error", "error", err, "subject", s.Subject, "queue", s.Queue)
			} else {
				slog.Error("async NATS error outside subscription", "error", err)
			}
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			// This handler means that max reconnect attempts have been exhausted.
			slog.Error(constants.ErrMsgNATSMaxReconnects)
			// In a full implementation, this would coordinate with graceful shutdown
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err, "nats_url", natsURL)
		return nil, err
	}

	slog.Info("Successfully connected to NATS", "connected_url", conn.ConnectedUrl())

	return &messagingRepository{
		conn: conn,
	}, nil
}

// Request sends a request message to the specified subject and waits for a response
func (r *messagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "nats.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("nats"),
			semconv.MessagingOperationTypeSend,
			semconv.MessagingDestinationName(subject),
			semconv.MessagingMessageBodySize(len(data)),
		),
	)
	defer span.End()

	if r.conn == nil {
		span.RecordError(constants.ErrNATSConnNotInit)
		span.SetStatus(codes.Error, constants.ErrNATSConnNotInit.Error())
		return nil, constants.ErrNATSConnNotInit
	}

	// Clamp timeout to the ctx deadline if it is shorter.
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	// Short-circuit if timeout is zero or negative — either the caller passed
	// an invalid duration or the ctx deadline was already past.
	if timeout <= 0 {
		span.RecordError(context.DeadlineExceeded)
		span.SetStatus(codes.Error, context.DeadlineExceeded.Error())
		return nil, context.DeadlineExceeded
	}

	natsMsg := nats.NewMsg(subject)
	natsMsg.Data = data
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(natsMsg.Header))

	msg, err := r.conn.RequestMsg(natsMsg, timeout)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, constants.ErrMsgNATSRequestFailed)
		return nil, fmt.Errorf("%s: %w", constants.ErrMsgNATSRequestFailed, err)
	}

	span.SetAttributes(messagingReplyBodySizeKey.Int(len(msg.Data)))
	span.SetStatus(codes.Ok, "")
	return msg.Data, nil
}

// Close closes the NATS connection gracefully
func (r *messagingRepository) Close() error {
	if r.conn != nil {
		err := r.conn.Drain()
		if err != nil {
			slog.Error("Failed to close NATS connection", "error", err)
			return err
		}
	}
	return nil
}

// HealthCheck verifies the NATS connection is healthy and responsive
func (r *messagingRepository) HealthCheck(ctx context.Context) error {
	if r.conn == nil {
		return constants.ErrNATSConnNotInit
	}

	// Check connection status
	if !r.conn.IsConnected() {
		return constants.ErrNATSConnNotActive
	}

	// Check if connection is closed or draining
	if r.conn.IsClosed() {
		return constants.ErrNATSConnClosed
	}

	if r.conn.IsDraining() {
		return constants.ErrNATSConnDraining
	}

	// Send a ping to verify responsiveness with timeout
	// Create a context with timeout for the health check
	healthCheckCtx, cancel := context.WithTimeout(ctx, constants.DefaultNATSTimeout)
	defer cancel()

	if err := r.conn.FlushWithContext(healthCheckCtx); err != nil {
		return fmt.Errorf("%s: %w", constants.ErrMsgNATSConnNotResponsive, err)
	}

	return nil
}
