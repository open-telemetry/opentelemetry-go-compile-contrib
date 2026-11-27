// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pgx

import (
	"context"

	"github.com/jackc/pgx/v5"

	"go.opentelemetry.io/otelc/pkg/hook"

	"go.opentelemetry.io/otelc-contrib/pkg/runtime"
)

func isEnabled() bool {
	return runtime.Instrumented(instrumentationKey)
}

// BeforeConnect hooks pgx.connect(ctx, config).
//
//nolint:revive // the parameters must mirror pgx.connect(ctx, config), which follow the HookContext
func BeforeConnect(_ hook.HookContext, _ context.Context, cfg *pgx.ConnConfig) {
	if !isEnabled() {
		return
	}

	if cfg == nil {
		return
	}

	cfg.Tracer = chainTracer(cfg.Tracer, newTracer())
}
