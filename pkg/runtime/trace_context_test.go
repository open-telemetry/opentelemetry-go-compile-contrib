// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestDefaultTraceAndSpanID(t *testing.T) {
	traceAndSpanIDFunc = defaultTraceAndSpanID
	traceID, spanID := GetTraceAndSpanID()
	assert.Equal(t, "", traceID)
	assert.Equal(t, "", spanID)
}

func TestRegisterTraceAndSpanIDFunc(t *testing.T) {
	original := traceAndSpanIDFunc
	defer func() { traceAndSpanIDFunc = original }()

	RegisterTraceAndSpanIDFunc(func() (string, string) {
		return "abc123", "def456"
	})

	traceID, spanID := GetTraceAndSpanID()
	assert.Equal(t, "abc123", traceID)
	assert.Equal(t, "def456", spanID)
}

func TestRegisterTraceAndSpanIDFunc_TraceOnly(t *testing.T) {
	original := traceAndSpanIDFunc
	defer func() { traceAndSpanIDFunc = original }()

	RegisterTraceAndSpanIDFunc(func() (string, string) {
		return "abc123", ""
	})

	traceID, spanID := GetTraceAndSpanID()
	assert.Equal(t, "abc123", traceID)
	assert.Equal(t, "", spanID)
}

func TestSpanFromContextWithoutGLS(t *testing.T) {
	var nilCtx context.Context
	assert.False(t, SpanFromContextWithoutGLS(nilCtx).SpanContext().IsValid())
	assert.False(t, SpanFromContextWithoutGLS(context.Background()).SpanContext().IsValid())
	assert.False(t, SpanFromContextWithoutGLS(context.TODO()).SpanContext().IsValid())

	type ctxKey struct{}
	withVal := context.WithValue(context.Background(), ctxKey{}, "value")
	assert.False(t, SpanFromContextWithoutGLS(withVal).SpanContext().IsValid())

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})
	withSpan := trace.ContextWithSpanContext(context.Background(), sc)
	assert.Equal(t, sc, SpanFromContextWithoutGLS(withSpan).SpanContext())
}

func TestWithoutGLSFallback(t *testing.T) {
	var nilCtx context.Context
	ctx := context.Background()
	assert.False(t, IsGLSFallbackDisabled(nilCtx))
	assert.False(t, IsGLSFallbackDisabled(ctx))

	marked := WithoutGLSFallback(ctx)
	assert.True(t, IsGLSFallbackDisabled(marked))
	assert.Same(t, marked, WithoutGLSFallback(marked))
	assert.False(t, IsGLSFallbackDisabled(ctx))
	assert.True(t, IsGLSFallbackDisabled(WithoutGLSFallback(nilCtx)))
}
