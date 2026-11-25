// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.opentelemetry.io/otelc/pkg/hook/hooktest"

	"go.opentelemetry.io/otelc-contrib/pkg/runtime"
)

func fakeSpan(traceByte, spanByte byte) trace.Span {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{traceByte},
		SpanID:     trace.SpanID{spanByte},
		TraceFlags: trace.FlagsSampled,
	})
	return trace.SpanFromContext(trace.ContextWithSpanContext(context.Background(), sc))
}

func TestAfterSpanFromContext(t *testing.T) {
	glsSpan := fakeSpan(1, 2)
	explicitSpan := fakeSpan(3, 4)

	runtime.RegisterSpanFromGLSFunc(func() trace.Span { return glsSpan })
	t.Cleanup(func() { runtime.RegisterSpanFromGLSFunc(func() trace.Span { return nil }) })

	t.Run("falls back to GLS when span is invalid", func(t *testing.T) {
		ictx := hooktest.NewMockHookContext(context.Background())
		ictx.ReturnVals = []any{noop.Span{}}

		afterSpanFromContext(ictx, noop.Span{})

		assert.Equal(t, glsSpan, ictx.GetReturnVal(0))
	})

	t.Run("falls back to GLS when ctx param is nil", func(t *testing.T) {
		var nilCtx context.Context
		ictx := hooktest.NewMockHookContext(nilCtx)
		ictx.ReturnVals = []any{noop.Span{}}

		afterSpanFromContext(ictx, noop.Span{})

		assert.Equal(t, glsSpan, ictx.GetReturnVal(0))
	})

	t.Run("skips GLS fallback when disabled on context", func(t *testing.T) {
		runtime.RegisterSpanFromGLSFunc(func() trace.Span {
			t.Fatal("GLS lookup should be skipped when disabled on context")
			return nil
		})
		t.Cleanup(func() { runtime.RegisterSpanFromGLSFunc(func() trace.Span { return glsSpan }) })

		ictx := hooktest.NewMockHookContext(runtime.WithoutGLSFallback(context.Background()))
		initial := noop.Span{}
		ictx.ReturnVals = []any{initial}

		afterSpanFromContext(ictx, initial)

		assert.Equal(t, initial, ictx.GetReturnVal(0))
	})

	t.Run("preserves valid span without reading GLS", func(t *testing.T) {
		ictx := hooktest.NewMockHookContext(context.Background())
		ictx.ReturnVals = []any{explicitSpan}

		afterSpanFromContext(ictx, explicitSpan)

		assert.Equal(t, explicitSpan, ictx.GetReturnVal(0))
	})
}
