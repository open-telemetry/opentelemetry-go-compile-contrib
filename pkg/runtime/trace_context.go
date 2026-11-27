// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type glsFallbackKey struct{}

// SpanFromContextWithoutGLS returns the span explicitly stored in ctx,
// bypassing the goroutine-local storage fallback in the trace.SpanFromContext hook.
func SpanFromContextWithoutGLS(ctx context.Context) trace.Span {
	if ctx == nil || ctx == context.Background() || ctx == context.TODO() {
		return noop.Span{}
	}
	return trace.SpanFromContext(WithoutGLSFallback(ctx))
}

// WithoutGLSFallback returns a copy of ctx that disables the goroutine-local
// storage fallback in the trace.SpanFromContext hook.
func WithoutGLSFallback(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if IsGLSFallbackDisabled(ctx) {
		return ctx
	}
	return context.WithValue(ctx, glsFallbackKey{}, true)
}

// IsGLSFallbackDisabled reports whether ctx disables the GLS fallback in the
// trace.SpanFromContext instrumentation hook.
func IsGLSFallbackDisabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(glsFallbackKey{}).(bool)
	return v
}

var (
	traceAndSpanIDFunc = defaultTraceAndSpanID
	spanFromGLSFunc    = defaultSpanFromGLS
)

func defaultTraceAndSpanID() (string, string) {
	return "", ""
}

func defaultSpanFromGLS() trace.Span {
	return nil
}

func GetTraceAndSpanID() (string, string) {
	return traceAndSpanIDFunc()
}

func RegisterTraceAndSpanIDFunc(f func() (string, string)) {
	traceAndSpanIDFunc = f
}

func GetSpanFromGLS() trace.Span {
	return spanFromGLSFunc()
}

func RegisterSpanFromGLSFunc(f func() trace.Span) {
	spanFromGLSFunc = f
}
