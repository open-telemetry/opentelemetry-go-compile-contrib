// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/log/otelclog"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/log/slog/otelcslog"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/runtime/otelcruntime"
)
