// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8s_client_go

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otelc/pkg/hook"
	"go.opentelemetry.io/otelc/pkg/hook/hooktest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func setupTestTracer(t *testing.T) (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return sr, tp
}

func TestK8SClientGoEnabler(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func(t *testing.T)
		expected bool
	}{
		{
			name: "enabled explicitly",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expected: true,
		},
		{
			name: "disabled explicitly",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expected: false,
		},
		{
			name: "not in enabled list",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "nethttp")
			},
			expected: false,
		},
		{
			name: "default enabled when no env set",
			setupEnv: func(t *testing.T) {
				// No environment variables set - should be enabled by default
			},
			expected: true,
		},
		{
			name: "enabled with multiple instrumentations",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "nethttp,k8s_client_go,grpc")
			},
			expected: true,
		},
		{
			name: "disabled with multiple instrumentations",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go,grpc")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv(t)

			enabler := k8SClientGoEnabler{}
			result := enabler.Enable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstrumentationConstants(t *testing.T) {
	assert.Equal(t,
		"go.opentelemetry.io/otelc-contrib/instrumentation/k8s.io/client-go/otelcclientgo",
		instrumentationName,
	)
	assert.Equal(t, "K8S_CLIENT_GO", instrumentationKey)
}

func TestBeforeProcessDeltas(t *testing.T) {
	for _, tt := range []struct {
		name        string
		setupEnv    func(t *testing.T)
		expectSpans bool
	}{
		{
			name: "instrumentation enabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectSpans: true,
		},
		{
			name: "instrumentation disabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectSpans: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initOnce = *new(sync.Once)

			tt.setupEnv(t)
			sr, _ := setupTestTracer(t)

			// A plain cache.ResourceEventHandlerFuncs is the real user handler
			// a non-shared informer (NewInformer, NewIndexerInformer, ...)
			// passes straight into processDeltas: it never goes through
			// AddEventHandlerWithOptions, so it still needs wrapping here.
			handler := cache.ResourceEventHandlerFuncs{}

			mockCtx := hooktest.NewMockHookContext(handler)
			beforeProcessDeltas(
				mockCtx,
				handler,
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
				[]cache.Delta{},
				false,
			)

			param := mockCtx.GetParam(0)
			if tt.expectSpans {
				wrapped, ok := param.(*k8SOtelEventHandler)
				require.True(t, ok, "non-shared-informer handler should be wrapped when enabled")
				assert.Equal(t, handler, wrapped.handler)
			} else {
				assert.Equal(t, handler, param, "handler should be untouched when disabled")
			}

			startedSpans := sr.Started()
			if tt.expectSpans {
				require.Len(t, startedSpans, 1, "should have started the objects.process span only")
			} else {
				require.Empty(t, startedSpans, "no spans should be emitted")
			}
		})
	}
}

// TestBeforeProcessDeltas_SharedInformerNotDoubleWrapped is the regression
// test for the opposite failure mode amazingakai flagged on #125: a
// *sharedIndexInformer passes itself as the handler to processDeltas only to
// fan events out to its own already-individually-wrapped listeners, so
// beforeProcessDeltas must leave it alone rather than wrapping it again.
func TestBeforeProcessDeltas_SharedInformerNotDoubleWrapped(t *testing.T) {
	initOnce = sync.Once{}
	t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
	setupTestTracer(t)

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListWithContextFunc: func(context.Context, metav1.ListOptions) (runtime.Object, error) {
				return nil, errors.New("unused")
			},
			WatchFuncWithContext: func(context.Context, metav1.ListOptions) (watch.Interface, error) {
				return nil, errors.New("unused")
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)
	handler, ok := informer.(cache.ResourceEventHandler)
	require.True(t, ok, "sharedIndexInformer must implement cache.ResourceEventHandler")

	mockCtx := hooktest.NewMockHookContext(handler)
	beforeProcessDeltas(
		mockCtx,
		handler,
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		[]cache.Delta{},
		false,
	)

	assert.Same(t, handler, mockCtx.GetParam(0), "a *sharedIndexInformer handler must not be wrapped again")
}

func TestAfterProcessDeltas(t *testing.T) {
	for _, tt := range []struct {
		name         string
		setupEnv     func(t *testing.T)
		setupContext func(*sdktrace.TracerProvider) hook.HookContext
		err          error
		validateSpan func(t *testing.T, spans []sdktrace.ReadOnlySpan)
	}{
		{
			name: "instrumentation enabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				span := spans[0]
				assert.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name: "error callback",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: errors.New("processing failed"),
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				span := spans[0]
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Contains(t, span.Status().Description, "processing failed")

				// Check that error was recorded
				events := span.Events()
				require.Len(t, events, 1)
				assert.Equal(t, "exception", events[0].Name)
			},
		},
		{
			name: "no data in context",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				return hooktest.NewMockHookContext()
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				// No span should be ended
				assert.Equal(t, 0, len(spans))
			},
		},
		{
			name: "disabled after span started still ends the span",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				// A span created by the before hook while enabled must still be
				// ended even if instrumentation is disabled before the after hook
				// runs, otherwise it leaks.
				require.Len(t, spans, 1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initOnce = *new(sync.Once)

			tt.setupEnv(t)
			sr, tp := setupTestTracer(t)

			mockCtx := tt.setupContext(tp)
			afterProcessDeltas(mockCtx, tt.err)

			spans := sr.Ended()
			if tt.validateSpan != nil {
				tt.validateSpan(t, spans)
			}
		})
	}
}

func TestBeforeProcessDeltasInBatch(t *testing.T) {
	for _, tt := range []struct {
		name        string
		setupEnv    func(t *testing.T)
		expectSpans bool
	}{
		{
			name: "instrumentation enabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectSpans: true,
		},
		{
			name: "instrumentation disabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectSpans: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initOnce = *new(sync.Once)

			tt.setupEnv(t)
			sr, _ := setupTestTracer(t)

			handler := cache.ResourceEventHandlerFuncs{}

			mockCtx := hooktest.NewMockHookContext(handler)
			beforeProcessDeltasInBatch(
				mockCtx,
				handler,
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
				[]cache.Delta{},
				false,
			)

			param := mockCtx.GetParam(0)
			if tt.expectSpans {
				wrapped, ok := param.(*k8SOtelEventHandler)
				require.True(t, ok, "non-shared-informer handler should be wrapped when enabled")
				assert.Equal(t, handler, wrapped.handler)
			} else {
				assert.Equal(t, handler, param, "handler should be untouched when disabled")
			}

			startedSpans := sr.Started()
			if tt.expectSpans {
				require.Len(t, startedSpans, 1, "should have started the objects.process span only")
			} else {
				require.Empty(t, startedSpans, "no spans should be emitted")
			}
		})
	}
}

// TestBeforeProcessDeltasInBatch_SharedInformerNotDoubleWrapped mirrors
// TestBeforeProcessDeltas_SharedInformerNotDoubleWrapped for the batch hook.
func TestBeforeProcessDeltasInBatch_SharedInformerNotDoubleWrapped(t *testing.T) {
	initOnce = sync.Once{}
	t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
	setupTestTracer(t)

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListWithContextFunc: func(context.Context, metav1.ListOptions) (runtime.Object, error) {
				return nil, errors.New("unused")
			},
			WatchFuncWithContext: func(context.Context, metav1.ListOptions) (watch.Interface, error) {
				return nil, errors.New("unused")
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)
	handler, ok := informer.(cache.ResourceEventHandler)
	require.True(t, ok, "sharedIndexInformer must implement cache.ResourceEventHandler")

	mockCtx := hooktest.NewMockHookContext(handler)
	beforeProcessDeltasInBatch(
		mockCtx,
		handler,
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		[]cache.Delta{},
		false,
	)

	assert.Same(t, handler, mockCtx.GetParam(0), "a *sharedIndexInformer handler must not be wrapped again")
}

func TestAfterProcessDeltasInBatch(t *testing.T) {
	for _, tt := range []struct {
		name         string
		setupEnv     func(t *testing.T)
		setupContext func(*sdktrace.TracerProvider) hook.HookContext
		err          error
		validateSpan func(t *testing.T, spans []sdktrace.ReadOnlySpan)
	}{
		{
			name: "instrumentation enabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				span := spans[0]
				assert.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name: "error callback",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: errors.New("processing failed"),
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				span := spans[0]
				assert.Equal(t, codes.Error, span.Status().Code)
				assert.Contains(t, span.Status().Description, "processing failed")

				// Check that error was recorded
				events := span.Events()
				require.Len(t, events, 1)
				assert.Equal(t, "exception", events[0].Name)
			},
		},
		{
			name: "no data in context",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				return hooktest.NewMockHookContext()
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				// No span should be ended
				assert.Equal(t, 0, len(spans))
			},
		},
		{
			name: "disabled after span started still ends the span",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			setupContext: func(tp *sdktrace.TracerProvider) hook.HookContext {
				testTracer := tp.Tracer(instrumentationName)
				_, span := testTracer.Start(context.Background(), "k8s.informer.objects.process", trace.WithSpanKind(trace.SpanKindInternal))

				mockCtx := hooktest.NewMockHookContext()
				mockCtx.SetKeyData("span", span)
				return mockCtx
			},
			err: nil,
			validateSpan: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				// A span created by the before hook while enabled must still be
				// ended even if instrumentation is disabled before the after hook
				// runs, otherwise it leaks.
				require.Len(t, spans, 1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initOnce = *new(sync.Once)

			tt.setupEnv(t)
			sr, tp := setupTestTracer(t)

			mockCtx := tt.setupContext(tp)
			afterProcessDeltasInBatch(mockCtx, tt.err)

			spans := sr.Ended()
			if tt.validateSpan != nil {
				tt.validateSpan(t, spans)
			}
		})
	}
}

// TestBeforeAddEventHandlerWithOptions verifies the handler argument is
// wrapped at registration time, and that the wrapper's span genuinely
// reflects the wrapped handler's own execution — the gap #80/#1183 reported
// against the processDeltas-level wrapping this replaces.
func TestBeforeAddEventHandlerWithOptions(t *testing.T) {
	for _, tt := range []struct {
		name       string
		setupEnv   func(t *testing.T)
		expectWrap bool
	}{
		{
			name: "instrumentation enabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectWrap: true,
		},
		{
			name: "instrumentation disabled",
			setupEnv: func(t *testing.T) {
				t.Setenv("OTEL_GO_DISABLED_INSTRUMENTATIONS", "k8s_client_go")
			},
			expectWrap: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initOnce = sync.Once{}

			tt.setupEnv(t)
			_, _ = setupTestTracer(t)

			handler := cache.ResourceEventHandlerFuncs{}
			options := cache.HandlerOptions{}

			mockCtx := hooktest.NewMockHookContext(nil, handler, options)
			beforeAddEventHandlerWithOptions(mockCtx, nil, handler, options)

			param := mockCtx.GetParam(1)
			if tt.expectWrap {
				wrapped, ok := param.(*k8SOtelEventHandler)
				require.True(t, ok, "handler should be wrapped when enabled")
				assert.Equal(t, handler, wrapped.handler)
			} else {
				assert.Equal(t, handler, param, "handler should be untouched when disabled")
			}
		})
	}
}

// TestBeforeAddEventHandlerWithOptions_NilHandler is the regression test for
// amazingakai's #125 review comment: a nil handler must be left alone
// instead of being wrapped, which would otherwise emit spans on every event
// right up until the wrapper's own call into the nil handler panics.
func TestBeforeAddEventHandlerWithOptions_NilHandler(t *testing.T) {
	initOnce = sync.Once{}
	t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
	_, _ = setupTestTracer(t)

	options := cache.HandlerOptions{}
	mockCtx := hooktest.NewMockHookContext(nil, nil, options)

	require.NotPanics(t, func() {
		beforeAddEventHandlerWithOptions(mockCtx, nil, nil, options)
	})
	assert.Nil(t, mockCtx.GetParam(1), "a nil handler must not be wrapped")
}

// TestBeforeAddEventHandlerWithOptions_SpanReflectsRealHandlerDuration is the
// direct regression test for #80/#1183: the span created for a dispatched
// event must cover the actual registered handler's execution time, not just
// the time client-go's own dispatch machinery takes to enqueue it.
func TestBeforeAddEventHandlerWithOptions_SpanReflectsRealHandlerDuration(t *testing.T) {
	initOnce = sync.Once{}
	t.Setenv("OTEL_GO_ENABLED_INSTRUMENTATIONS", "k8s_client_go")
	sr, _ := setupTestTracer(t)

	const handlerDelay = 20 * time.Millisecond
	realHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			time.Sleep(handlerDelay)
		},
	}

	mockCtx := hooktest.NewMockHookContext(nil, realHandler, cache.HandlerOptions{})
	beforeAddEventHandlerWithOptions(mockCtx, nil, realHandler, cache.HandlerOptions{})

	wrapped, ok := mockCtx.GetParam(1).(*k8SOtelEventHandler)
	require.True(t, ok)

	wrapped.OnAdd(&corev1.Pod{}, false)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	duration := spans[0].EndTime().Sub(spans[0].StartTime())
	assert.GreaterOrEqual(t, duration, handlerDelay,
		"span duration should reflect the handler's own execution time, not just dispatch")
}
