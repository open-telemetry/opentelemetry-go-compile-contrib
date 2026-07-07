// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hook

import (
	_ "unsafe"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otelc/pkg/hook"
)

//go:linkname traceContextAddSpan go.opentelemetry.io/otel/sdk/trace.traceContextAddSpan
func traceContextAddSpan(span trace.Span)

func newRecordingSpanAfter(ictx hook.HookContext, span interface{}) {
	addSpanToGls(span)
}

func newNonRecordingSpanAfter(ictx hook.HookContext, span interface{}) {
	addSpanToGls(span)
}

func addSpanToGls(span interface{}) {
	if span != nil {
		if s, ok := span.(trace.Span); ok {
			traceContextAddSpan(s)
		}
	}
}
