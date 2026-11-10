// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/github.com/sirupsen/logrus/otelclogrus"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/runtime/otelcruntime"
)
