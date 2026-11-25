// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"go.opentelemetry.io/otelc-contrib/test/testutil"
)

const pgxScope = "go.opentelemetry.io/otelc-contrib/instrumentation/github.com/jackc/pgx/v5/otelcpgx"

func TestPGXClient(t *testing.T) {
	t.Parallel()
	testutil.Build(t, "", "pgxclient", "go", "build", "-a")

	if runtime.GOOS == "windows" {
		t.Skip("postgres testcontainer not supported on windows")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	host, port := startPostgresContainer(t)
	connStr := fmt.Sprintf("postgres://otelc:otelc@%s:%d/testdb?sslmode=disable", host, port)

	t.Run("Connect", func(t *testing.T) {
		f := testutil.NewTestFixture(t)
		output := f.Run("pgxclient", "-mode=connect", "-conn="+connStr)
		require.Contains(t, output, "pgx.Connect successful")

		spans := pgxSpans(f.Traces())
		require.NotEmpty(t, spans)

		querySpan := testutil.RequireSpan(t, f.Traces(),
			testutil.IsClient,
			testutil.HasAttribute("db.operation.name", "SELECT"),
		)
		require.Equal(t, ptrace.SpanKindClient, querySpan.Kind())
		testutil.RequireAttribute(t, querySpan, "db.system.name", "postgresql")
		testutil.RequireAttribute(t, querySpan, "db.namespace", "testdb")
		testutil.RequireAttribute(t, querySpan, "server.address", host)
		testutil.RequireAttribute(t, querySpan, "server.port", int64(port))
	})

	t.Run("Pool", func(t *testing.T) {
		f := testutil.NewTestFixture(t)
		output := f.Run("pgxclient", "-mode=pool", "-conn="+connStr)
		require.Contains(t, output, "pgxpool.NewWithConfig successful")

		spans := pgxSpans(f.Traces())
		require.NotEmpty(t, spans)

		querySpan := testutil.RequireSpan(t, f.Traces(),
			testutil.IsClient,
			testutil.HasAttribute("db.operation.name", "SELECT"),
		)
		require.Equal(t, ptrace.SpanKindClient, querySpan.Kind())
		testutil.RequireAttribute(t, querySpan, "db.system.name", "postgresql")
		testutil.RequireAttribute(t, querySpan, "db.namespace", "testdb")
		testutil.RequireAttribute(t, querySpan, "server.address", host)
		testutil.RequireAttribute(t, querySpan, "server.port", int64(port))
	})
}

func startPostgresContainer(t *testing.T) (string, int) {
	t.Helper()

	ctr, err := testcontainers.Run(t.Context(), "postgres:16-alpine",
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "otelc",
			"POSTGRES_PASSWORD": "otelc",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)

	host, err := ctr.Host(t.Context())
	require.NoError(t, err)
	mapped, err := ctr.MappedPort(t.Context(), "5432/tcp")
	require.NoError(t, err)
	return host, int(mapped.Num())
}

func pgxSpans(td ptrace.Traces) []ptrace.Span {
	var spans []ptrace.Span
	for i := range td.ResourceSpans().Len() {
		rs := td.ResourceSpans().At(i)
		for j := range rs.ScopeSpans().Len() {
			ss := rs.ScopeSpans().At(j)
			if ss.Scope().Name() != pgxScope {
				continue
			}
			for k := range ss.Spans().Len() {
				spans = append(spans, ss.Spans().At(k))
			}
		}
	}
	return spans
}
