// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/github.com/linode/linodego/v2/otelclinodego"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp/client"
)
