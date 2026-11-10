// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelcaws"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init"
)
