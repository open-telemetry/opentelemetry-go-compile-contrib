// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelcmongo"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelcmongo"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init"
)
