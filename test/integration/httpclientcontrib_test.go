// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"go.opentelemetry.io/otelc-contrib/test/testutil"
)

const (
	otelHTTPScope  = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelcHTTPScope = "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp"
)

// TestHTTPClientWithOtelhttpTransport runs the httpclientcontrib app. The
// front handler makes one call through otelhttp.NewTransport and one call
// through a plain http.Client. Each call must carry exactly one client span:
// otelhttp's for the wrapped transport, otelc's for the plain client. otelc
// must not add a second client span under otelhttp.NewTransport.
func TestHTTPClientWithOtelhttpTransport(t *testing.T) {
	t.Parallel()
	testutil.Build(t, "", "httpclientcontrib", "go", "build", "-a")

	f := testutil.NewTestFixture(t)
	frontPort := testutil.FreePort(t)
	backPort := testutil.FreePort(t)

	f.Start("httpclientcontrib",
		fmt.Sprintf("-front-port=%d", frontPort),
		fmt.Sprintf("-back-port=%d", backPort))
	testutil.WaitForTCP(t, fmt.Sprintf("127.0.0.1:%d", frontPort))
	testutil.WaitForTCP(t, fmt.Sprintf("127.0.0.1:%d", backPort))

	url := fmt.Sprintf("http://127.0.0.1:%d/call", frontPort)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Front server span, two client spans, and two backend server spans.
	f.WaitForSpans(5)

	td := f.Traces()
	f.RequireTraceCount(1)
	f.RequireSpansPerTrace(5)

	frontServer := requireOneScopedSpan(t, td, "", testutil.IsServer, testutil.HasName("GET /call"))
	otelhttpClient := requireOneScopedSpan(t, td, otelHTTPScope, testutil.IsClient)
	otelcClient := requireOneScopedSpan(t, td, otelcHTTPScope, testutil.IsClient)
	otelhttpBack := requireOneScopedSpan(t, td, "", testutil.IsServer, testutil.HasName("GET /back-otelhttp"))
	plainBack := requireOneScopedSpan(t, td, "", testutil.IsServer, testutil.HasName("GET /back-plain"))

	require.True(t, frontServer.ParentSpanID().IsEmpty(), "front server span must be the trace root")

	// Each client span is a child of the front server span.
	require.Equal(t, frontServer.SpanID(), otelhttpClient.ParentSpanID(),
		"otelhttp client span must be a child of the front server span")
	require.Equal(t, frontServer.SpanID(), otelcClient.ParentSpanID(),
		"otelc client span must be a child of the front server span")

	// Each backend server span follows the traceparent of its call, so the
	// downstream server span is a child of the client span that was in
	// charge of that call.
	require.Equal(t, otelhttpClient.SpanID(), otelhttpBack.ParentSpanID(),
		"backend span behind otelhttp must be a child of the otelhttp client span")
	require.Equal(t, otelcClient.SpanID(), plainBack.ParentSpanID(),
		"backend span behind the plain client must be a child of the otelc client span")

	testutil.RequireNoDanglingParents(t, td)
}

// requireOneScopedSpan returns the only span in the given instrumentation scope
// that matches all matchers. An empty scope matches every scope.
func requireOneScopedSpan(t *testing.T, td ptrace.Traces, scope string, matchers ...testutil.SpanMatcher) ptrace.Span {
	t.Helper()
	var matches []ptrace.Span
	rs := td.ResourceSpans()
	for i := range rs.Len() {
		ss := rs.At(i).ScopeSpans()
		for j := range ss.Len() {
			scopeSpans := ss.At(j)
			spans := scopeSpans.Spans()
			for k := range spans.Len() {
				s := spans.At(k)
				if scope != "" && scopeSpans.Scope().Name() != scope {
					continue
				}
				match := true
				for _, m := range matchers {
					if !m(s) {
						match = false
						break
					}
				}
				if match {
					matches = append(matches, s)
				}
			}
		}
	}
	require.Len(t, matches, 1, "expected exactly one span in scope %q matching the matchers", scope)
	return matches[0]
}
