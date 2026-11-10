// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/trace/otelctrace"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp/client"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp/server"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/runtime/otelcruntime"
)
