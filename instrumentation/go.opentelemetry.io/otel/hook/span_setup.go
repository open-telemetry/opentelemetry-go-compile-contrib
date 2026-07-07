// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hook

import (
	_ "unsafe"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otelc/pkg/hook"
)

//go:linkname traceContextDelSpan go.opentelemetry.io/otel/sdk/trace.traceContextDelSpan
func traceContextDelSpan(span trace.Span)

func nonRecordingSpanEndBefore(ictx hook.HookContext, span interface{}, options ...interface{}) {
	deleteFromGls(span)
}

func recordingSpanEndBefore(ictx hook.HookContext, span interface{}, options ...interface{}) {
	deleteFromGls(span)
}

func deleteFromGls(span interface{}) {
	if span != nil {
		if s, ok := span.(trace.Span); ok {
			traceContextDelSpan(s)
		}
	}
}
