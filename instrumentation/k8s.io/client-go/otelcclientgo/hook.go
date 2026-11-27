// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8s_client_go

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/tools/cache"

	"go.opentelemetry.io/otelc/pkg/hook"

	"go.opentelemetry.io/otelc-contrib/instrumentation/k8s.io/client-go/otelcclientgo/semconv"
	"go.opentelemetry.io/otelc-contrib/pkg/runtime"
)

const (
	instrumentationName = "go.opentelemetry.io/otelc-contrib/instrumentation/k8s.io/client-go/otelcclientgo"
	instrumentationKey  = "K8S_CLIENT_GO"
)

var (
	logger   = runtime.Logger()
	tracer   trace.Tracer
	initOnce sync.Once
)

// k8SClientGoEnabler controls whether library instrumentation is enabled
type k8SClientGoEnabler struct{}

func (g k8SClientGoEnabler) Enable() bool {
	return runtime.Instrumented(instrumentationKey)
}

var k8SEnabler = k8SClientGoEnabler{}

func initInstrumentation() {
	initOnce.Do(func() {
		tracer = otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(runtime.ModuleVersion()),
		)
		logger.Info("K8S client-go instrumentation initialized")
	})
}

func beforeProcessDeltas(
	ictx hook.HookContext,
	_ cache.ResourceEventHandler,
	_ cache.Store,
	deltas cache.Deltas,
	isInInitialList bool,
) {
	if !runtime.Ready() {
		return
	}
	if !k8SEnabler.Enable() {
		logger.Debug("K8S client-go instrumentation disabled")
		return
	}
	initInstrumentation()

	objsInfo := semconv.K8SObjectsInfo{
		Count:           len(deltas),
		IsInInitialList: isInInitialList,
	}
	attrs := semconv.K8SObjectsInfoTraceAttrs(objsInfo)

	// This span covers the cache-mutation dispatch itself (Store.Update/
	// Add/Delete for this batch), not any registered handler's own
	// execution: the handler argument here is the shared informer's
	// internal processor, which only enqueues notifications onto each
	// listener's channel and returns almost immediately. Each listener's
	// own handler is wrapped separately, at AddEventHandlerWithOptions
	// registration time (see beforeAddEventHandlerWithOptions), so its
	// span reflects the handler's real, later, asynchronous execution.
	spanName := "k8s.informer.objects.process"
	//nolint:spancheck // ended by afterProcessDeltas via the "span" key data, not here
	_, span := tracer.Start(context.Background(),
		spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)

	ictx.SetKeyData("span", span)
}

func afterProcessDeltas(ictx hook.HookContext, err error) {
	span, ok := ictx.GetKeyData("span").(trace.Span)
	if !ok || span == nil {
		// The logger is nil for a call skipped during package initialization.
		if !runtime.Ready() {
			return
		}
		logger.Debug("afterProcessDeltas: no span from before hook")
		return
	}
	defer span.End()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

func beforeProcessDeltasInBatch(
	ictx hook.HookContext,
	_ cache.ResourceEventHandler,
	_ cache.Store,
	deltas []cache.Delta,
	isInInitialList bool,
) {
	if !runtime.Ready() {
		return
	}
	if !k8SEnabler.Enable() {
		logger.Debug("K8S client-go instrumentation disabled")
		return
	}
	initInstrumentation()

	objsInfo := semconv.K8SObjectsInfo{
		Count:           len(deltas),
		IsInInitialList: isInInitialList,
	}
	attrs := semconv.K8SObjectsInfoTraceAttrs(objsInfo)

	// See beforeProcessDeltas: this span covers cache-mutation dispatch
	// only, not any registered handler's own execution.
	spanName := "k8s.informer.objects.process"
	//nolint:spancheck // ended by afterProcessDeltasInBatch via the "span" key data, not here
	_, span := tracer.Start(context.Background(),
		spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)

	ictx.SetKeyData("span", span)
}

func afterProcessDeltasInBatch(ictx hook.HookContext, err error) {
	span, ok := ictx.GetKeyData("span").(trace.Span)
	if !ok || span == nil {
		// The logger is nil for a call skipped during package initialization.
		if !runtime.Ready() {
			return
		}
		logger.Debug("afterProcessDeltasInBatch: no span from before hook")
		return
	}
	defer span.End()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// beforeAddEventHandlerWithOptions wraps a caller's handler at the point
// where it actually gets registered with a *sharedIndexInformer, so the
// wrapper's OnAdd/OnUpdate/OnDelete run exactly where and when client-go
// invokes the real handler: inside that listener's own dispatch goroutine,
// not inside processDeltas/processDeltasInBatch (see beforeProcessDeltas).
// AddEventHandler and AddEventHandlerWithResyncPeriod both delegate to
// AddEventHandlerWithOptions internally, so this one hook covers every
// registration path. The receiver type is unexported in client-go, so it is
// received as any and otherwise unused.
func beforeAddEventHandlerWithOptions(
	ictx hook.HookContext,
	_ any,
	handler cache.ResourceEventHandler,
	_ cache.HandlerOptions,
) {
	if !runtime.Ready() {
		return
	}
	if !k8SEnabler.Enable() {
		return
	}
	initInstrumentation()

	ictx.SetParam(1, newK8SOtelEventHandler(handler))
}
