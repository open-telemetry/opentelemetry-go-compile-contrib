# Runtime Configuration

This document describes runtime environment variables supported by the injected `otelc` runtime and instrumentation packages.

For standard OpenTelemetry SDK environment variables (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER`, `OTEL_SERVICE_NAME`, etc.), see the [OTel SDK environment variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) specification.

## Root Span Filtering

The injected runtime can suppress unwanted traces at sampling time. Set
`OTELC_DROP_ROOT_CLIENT_PRODUCER=true` to drop parentless client and producer
spans. Set `OTELC_EXCLUDED_SERVER_PATHS=/healthz,/readyz` to drop root server
spans whose initial `url.path` matches one of those paths. The path match is
exact and case-sensitive; spaces around comma-separated entries are ignored.

Both options are disabled by default. When enabled, local children of any
unsampled span are also dropped. With `OTEL_TRACES_SAMPLER=always_on`, this
changes the SDK's usual behavior for children of unsampled local parents.
Spans with a valid remote parent follow the configured SDK sampler. The filter
supports the SDK's `OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG` options.

## Instrumentation Selection

Every compiled-in instrumentation can be enabled or disabled at runtime via:

- `OTEL_GO_ENABLED_INSTRUMENTATIONS` — comma-separated allowlist
- `OTEL_GO_DISABLED_INSTRUMENTATIONS` — comma-separated denylist

See [Runtime Selection](troubleshooting.md#runtime-selection) in the troubleshooting guide for details and examples.

## Goroutine-Local Storage (GLS) Limits

The OTel SDK GLS fallback bounds its per-goroutine span chain and shared span end-state map via:

- `OTEL_GLS_MAX_SPANS` (default: `1000`)
- `OTEL_GLS_MAX_SPAN_STATES` (default: `100000`)

See [Injected trace context holder](instrument-guide.md#2-injected-trace-context-holder) for details on how these limits behave.
