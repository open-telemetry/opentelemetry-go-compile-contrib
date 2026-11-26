// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pgx

import (
	"github.com/jackc/pgx/v5"

	"go.opentelemetry.io/otelc/pkg/hook"

	"go.opentelemetry.io/otelc-contrib/pkg/runtime"
)

const (
	minConnectParams = 2
)

func isEnabled() bool {
	return runtime.Instrumented(instrumentationKey)
}

// BeforeConnect hooks pgx.connect(ctx, config)
func BeforeConnect(call hook.HookContext) {
	if !isEnabled() {
		return
	}

	if call.GetParamCount() < minConnectParams {
		return
	}
	cfg, ok := call.GetParam(1).(*pgx.ConnConfig)
	if !ok || cfg == nil {
		return
	}

	cfg.Tracer = chainTracer(cfg.Tracer, newTracer())
}
