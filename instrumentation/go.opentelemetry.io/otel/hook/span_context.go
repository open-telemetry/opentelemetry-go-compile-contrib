// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package hook

import (
	_ "unsafe"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otelc/pkg/hook"
)

//go:linkname spanFromGLS go.opentelemetry.io/otel/sdk/trace.spanFromGLS
func spanFromGLS() trace.Span

func spanFromContextOnExit(ictx hook.HookContext, span trace.Span) {
	if !span.SpanContext().IsValid() {
		glsSpan := spanFromGLS()
		if glsSpan != nil {
			ictx.SetReturnVal(0, glsSpan)
		}
	}
}
