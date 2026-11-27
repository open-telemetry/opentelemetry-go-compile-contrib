// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pgx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otelc-contrib/pkg/runtime"
)

const (
	instrumentationName = "go.opentelemetry.io/otelc-contrib/instrumentation/github.com/jackc/pgx/v5/otelcpgx"
	instrumentationKey  = "PGX"
	expectedConnAttrs   = 5
)

type tracer struct {
	tr trace.Tracer
}

func newTracer() *tracer {
	return &tracer{
		tr: otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(runtime.ModuleVersion()),
		),
	}
}

func skipLeadingComments(query string) string {
	s := strings.TrimSpace(query)
	for {
		switch {
		case strings.HasPrefix(s, "--"):
			end := strings.IndexAny(s, "\r\n")
			if end < 0 {
				return ""
			}
			s = strings.TrimSpace(s[end+1:])
		case strings.HasPrefix(s, "/*"):
			depth := 1
			i := 2
			chars := []rune(s)
			n := len(chars)
			for i < n && depth > 0 {
				switch {
				case i+1 < n && chars[i] == '/' && chars[i+1] == '*':
					depth++
					i += 2
				case i+1 < n && chars[i] == '*' && chars[i+1] == '/':
					depth--
					i += 2
				default:
					i++
				}
			}
			if depth > 0 {
				return ""
			}
			s = strings.TrimSpace(string(chars[i:]))
		default:
			return s
		}
	}
}

// operationName returns the uppercased first SQL keyword (e.g. SELECT, INSERT, UPDATE).
// It stops the keyword at the first character that is not a letter.
func operationName(query string) string {
	trimmed := skipLeadingComments(query)
	if trimmed == "" {
		return ""
	}
	end := strings.IndexFunc(trimmed, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if end == 0 {
		return ""
	}
	if end > 0 {
		trimmed = trimmed[:end]
	}
	return strings.ToUpper(trimmed)
}

func configAttributes(cfg *pgx.ConnConfig) []attribute.KeyValue {
	if cfg == nil {
		return []attribute.KeyValue{semconv.DBSystemNamePostgreSQL}
	}

	attrs := make([]attribute.KeyValue, 0, expectedConnAttrs)
	attrs = append(attrs, semconv.DBSystemNamePostgreSQL)

	if cfg.Database != "" {
		attrs = append(attrs, semconv.DBNamespace(cfg.Database))
	}
	if cfg.Host != "" {
		host := cfg.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		attrs = append(attrs, semconv.ServerAddress(host))
		attrs = append(attrs, semconv.NetworkTransportTCP)
	}
	if cfg.Port > 0 {
		attrs = append(attrs, semconv.ServerPort(int(cfg.Port)))
	}

	return attrs
}

func connAttributes(conn *pgx.Conn) []attribute.KeyValue {
	if conn == nil {
		return []attribute.KeyValue{semconv.DBSystemNamePostgreSQL}
	}
	return configAttributes(conn.Config())
}

func (t *tracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	if !isEnabled() {
		return ctx
	}

	op := operationName(data.SQL)
	spanName := op
	if spanName == "" {
		spanName = "DB Query"
	}
	if conn != nil && conn.Config() != nil && conn.Config().Database != "" {
		spanName = fmt.Sprintf("%s %s", spanName, conn.Config().Database)
	}

	attrs := connAttributes(conn)
	if data.SQL != "" {
		attrs = append(attrs, semconv.DBQueryText(sanitizeSQL(data.SQL)))
	}
	if op != "" {
		attrs = append(attrs, semconv.DBOperationName(op))
	}

	//nolint:spancheck // span is ended in TraceQueryEnd
	ctx, _ = t.tr.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return ctx
}

func (*tracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	if !isEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}

	span.End()
}

func (t *tracer) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	_ pgx.TraceBatchStartData,
) context.Context {
	if !isEnabled() {
		return ctx
	}

	spanName := "BATCH"
	if conn != nil && conn.Config() != nil && conn.Config().Database != "" {
		spanName = fmt.Sprintf("BATCH %s", conn.Config().Database)
	}

	attrs := connAttributes(conn)
	attrs = append(attrs, semconv.DBOperationName("BATCH"))

	//nolint:spancheck // span is ended in TraceBatchEnd
	ctx, _ = t.tr.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return ctx
}

func (*tracer) TraceBatchQuery(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchQueryData,
) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
}

func (*tracer) TraceBatchEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchEndData,
) {
	if !isEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}

	span.End()
}

func (t *tracer) TraceCopyFromStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	if !isEnabled() {
		return ctx
	}

	spanName := "COPY"
	if conn != nil && conn.Config() != nil && conn.Config().Database != "" {
		spanName = fmt.Sprintf("COPY %s", conn.Config().Database)
	}

	attrs := connAttributes(conn)
	attrs = append(attrs, semconv.DBOperationName("COPY"))
	if len(data.TableName) > 0 {
		attrs = append(attrs, semconv.DBCollectionName(data.TableName.Sanitize()))
	}

	//nolint:spancheck // span is ended in TraceCopyFromEnd
	ctx, _ = t.tr.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return ctx
}

func (*tracer) TraceCopyFromEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceCopyFromEndData,
) {
	if !isEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}

	span.End()
}

func (t *tracer) TracePrepareStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	if !isEnabled() {
		return ctx
	}

	spanName := "PREPARE"
	if conn != nil && conn.Config() != nil && conn.Config().Database != "" {
		spanName = fmt.Sprintf("PREPARE %s", conn.Config().Database)
	}

	attrs := connAttributes(conn)
	attrs = append(attrs, semconv.DBOperationName("PREPARE"))
	if data.SQL != "" {
		attrs = append(attrs, semconv.DBQueryText(sanitizeSQL(data.SQL)))
	}

	//nolint:spancheck // span is ended in TracePrepareEnd
	ctx, _ = t.tr.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return ctx
}

func (*tracer) TracePrepareEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TracePrepareEndData,
) {
	if !isEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}

	span.End()
}

func (*tracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	return ctx
}

func (*tracer) TraceConnectEnd(_ context.Context, _ pgx.TraceConnectEndData) {
}

// chainedTracer chains multiple tracers so custom user tracers continue to fire.
type chainedTracer struct {
	tracers []pgx.QueryTracer
}

func hasTracer(t pgx.QueryTracer) bool {
	if _, ok := t.(*tracer); ok {
		return true
	}
	if ct, ok := t.(*chainedTracer); ok {
		return slices.ContainsFunc(ct.tracers, hasTracer)
	}
	return false
}

func chainTracer(existing, otelTracer pgx.QueryTracer) pgx.QueryTracer {
	if existing == nil {
		return otelTracer
	}
	if hasTracer(existing) {
		return existing
	}
	if ct, ok := existing.(*chainedTracer); ok {
		return &chainedTracer{tracers: append(ct.tracers, otelTracer)}
	}
	return &chainedTracer{tracers: []pgx.QueryTracer{existing, otelTracer}}
}

type (
	queryCtxKey   struct{}
	batchCtxKey   struct{}
	copyCtxKey    struct{}
	prepareCtxKey struct{}
	connectCtxKey struct{}
)

func (c *chainedTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	contexts := make([]context.Context, len(c.tracers))
	for i, tr := range c.tracers {
		//nolint:fatcontext // nested context chaining across tracers is intentional
		ctx = tr.TraceQueryStart(ctx, conn, data)
		contexts[i] = ctx
	}
	return context.WithValue(ctx, queryCtxKey{}, contexts)
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	contexts, _ := ctx.Value(queryCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		tracerCtx := ctx
		if i < len(contexts) && contexts[i] != nil {
			tracerCtx = contexts[i]
		}
		tr.TraceQueryEnd(tracerCtx, conn, data)
	}
}

func (c *chainedTracer) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	contexts := make([]context.Context, len(c.tracers))
	for i, tr := range c.tracers {
		if bt, ok := tr.(pgx.BatchTracer); ok {
			//nolint:fatcontext // nested context chaining across tracers is intentional
			ctx = bt.TraceBatchStart(ctx, conn, data)
		}
		contexts[i] = ctx
	}
	return context.WithValue(ctx, batchCtxKey{}, contexts)
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TraceBatchQuery(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchQueryData,
) {
	contexts, _ := ctx.Value(batchCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		if bt, ok := tr.(pgx.BatchTracer); ok {
			tracerCtx := ctx
			if i < len(contexts) && contexts[i] != nil {
				tracerCtx = contexts[i]
			}
			bt.TraceBatchQuery(tracerCtx, conn, data)
		}
	}
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TraceBatchEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchEndData,
) {
	contexts, _ := ctx.Value(batchCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		if bt, ok := tr.(pgx.BatchTracer); ok {
			tracerCtx := ctx
			if i < len(contexts) && contexts[i] != nil {
				tracerCtx = contexts[i]
			}
			bt.TraceBatchEnd(tracerCtx, conn, data)
		}
	}
}

func (c *chainedTracer) TraceCopyFromStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromStartData,
) context.Context {
	contexts := make([]context.Context, len(c.tracers))
	for i, tr := range c.tracers {
		if ct, ok := tr.(pgx.CopyFromTracer); ok {
			//nolint:fatcontext // nested context chaining across tracers is intentional
			ctx = ct.TraceCopyFromStart(ctx, conn, data)
		}
		contexts[i] = ctx
	}
	return context.WithValue(ctx, copyCtxKey{}, contexts)
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TraceCopyFromEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceCopyFromEndData,
) {
	contexts, _ := ctx.Value(copyCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		if ct, ok := tr.(pgx.CopyFromTracer); ok {
			tracerCtx := ctx
			if i < len(contexts) && contexts[i] != nil {
				tracerCtx = contexts[i]
			}
			ct.TraceCopyFromEnd(tracerCtx, conn, data)
		}
	}
}

func (c *chainedTracer) TracePrepareStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareStartData,
) context.Context {
	contexts := make([]context.Context, len(c.tracers))
	for i, tr := range c.tracers {
		if pt, ok := tr.(pgx.PrepareTracer); ok {
			//nolint:fatcontext // nested context chaining across tracers is intentional
			ctx = pt.TracePrepareStart(ctx, conn, data)
		}
		contexts[i] = ctx
	}
	return context.WithValue(ctx, prepareCtxKey{}, contexts)
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TracePrepareEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TracePrepareEndData,
) {
	contexts, _ := ctx.Value(prepareCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		if pt, ok := tr.(pgx.PrepareTracer); ok {
			tracerCtx := ctx
			if i < len(contexts) && contexts[i] != nil {
				tracerCtx = contexts[i]
			}
			pt.TracePrepareEnd(tracerCtx, conn, data)
		}
	}
}

func (c *chainedTracer) TraceConnectStart(
	ctx context.Context,
	data pgx.TraceConnectStartData,
) context.Context {
	contexts := make([]context.Context, len(c.tracers))
	for i, tr := range c.tracers {
		if cnt, ok := tr.(pgx.ConnectTracer); ok {
			//nolint:fatcontext // nested context chaining across tracers is intentional
			ctx = cnt.TraceConnectStart(ctx, data)
		}
		contexts[i] = ctx
	}
	return context.WithValue(ctx, connectCtxKey{}, contexts)
}

//nolint:contextcheck // per-tracer context restored from matching Start call
func (c *chainedTracer) TraceConnectEnd(
	ctx context.Context,
	data pgx.TraceConnectEndData,
) {
	contexts, _ := ctx.Value(connectCtxKey{}).([]context.Context)
	for i, tr := range c.tracers {
		if cnt, ok := tr.(pgx.ConnectTracer); ok {
			tracerCtx := ctx
			if i < len(contexts) && contexts[i] != nil {
				tracerCtx = contexts[i]
			}
			cnt.TraceConnectEnd(tracerCtx, data)
		}
	}
}
