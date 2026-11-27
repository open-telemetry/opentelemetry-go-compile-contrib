# Troubleshooting Instrumentation

This guide covers common runtime problems and inherent limitations of compile-time hooks in `opentelemetry-go-compile-contrib`.

For issues with the compiler toolchain, `-toolexec` build failures, rule matching decisions, or compiler build cache corruption, see the [upstream compiler troubleshooting guide](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation/blob/main/docs/troubleshooting.md).

## Instrumented but no spans appear

Since `otelc` injects the OpenTelemetry SDK initialization automatically, instrumented binaries emit telemetry by default. If spans do not appear in your backend, the issue is typically exporter configuration, not instrumentation:

- Verify the exporter endpoint: `OTEL_EXPORTER_OTLP_ENDPOINT`
- Verify the exporter protocol: `OTEL_EXPORTER_OTLP_PROTOCOL` (default: `http/protobuf`)
- Check sampler settings: `OTEL_TRACES_SAMPLER` (default: `parentbased_always_on`)

The [OTel SDK environment variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) specification documents all available knobs.

## Runtime Selection

Every instrumentation can be toggled without rebuilding using environment variables:

- **`OTEL_GO_ENABLED_INSTRUMENTATIONS`** — comma-separated allowlist. When set, only the listed instrumentations run; all others are turned off.
- **`OTEL_GO_DISABLED_INSTRUMENTATIONS`** — comma-separated denylist. The listed instrumentations are turned off. Applied after the enabled list.

Names are lowercase and matched case-insensitively, e.g. `nethttp,grpc`. When neither variable is set, all instrumentations compiled into the binary run (the default). When both are set, the enabled allowlist is applied first, then the disabled list removes entries from it.

```bash
# Run only net/http and gRPC instrumentation
OTEL_GO_ENABLED_INSTRUMENTATIONS=nethttp,grpc ./myapp

# Run everything except net/http
OTEL_GO_DISABLED_INSTRUMENTATIONS=nethttp ./myapp
```

> [!NOTE]
> These variables only gate instrumentations already compiled into the binary. To control what is compiled in, use `otel.instrumentation.go` (see [Usage in README](../README.md#usage)).

## Duplicate Spans with `otelhttp`, `otelgin` and `otelgrpc`

An application can use wrappers from [opentelemetry-go-contrib](https://github.com/open-telemetry/opentelemetry-go-contrib) and also be built with `otelc`. Both layers can then record the same request. The result depends on the path:

| Path | Result |
| --- | --- |
| `http.Client` with `otelhttp.NewTransport` over `*http.Transport` | One CLIENT span. `otelhttp` owns the span. `otelc` steps aside when the request context carries a recording `otelhttp` CLIENT span. |
| `net/http` server with `otelhttp.NewHandler`, or gin with `otelgin.Middleware` | Two SERVER spans per request. `otelc` starts its span at the outer `net/http` layer, outside the app handler. The wrapper then starts a second SERVER span. See below for how the two relate. |
| gRPC server or client with `otelgrpc` stats handlers | Two SERVER spans or two CLIENT spans per hop. Both are exported. On the client, `otelc`'s span is the parent of `otelgrpc`'s span. On the server, the two spans nest like the HTTP server spans. |

On the server side, for HTTP and gRPC, the relation between the two SERVER spans depends on the incoming request. Without an incoming `traceparent`, the wrapper's span is a child of `otelc`'s span. With an incoming `traceparent` in the HTTP headers or the gRPC metadata, the wrapper extracts it again and starts its span from the remote parent. The two spans are then siblings.

For the server paths and gRPC, remove the contrib wrapper and rely on `otelc` for that path. This gives one span per hop.

If you must keep the wrapper, disable `otelc` for that protocol with `OTEL_GO_DISABLED_INSTRUMENTATIONS`. The `net/http` client and server instrumentation share the key `nethttp`, and gRPC uses `grpc`. Each key covers the client and the server side, so it also disables `otelc` for calls that the wrapper does not cover. See [Runtime Selection](#runtime-selection).

Disabling the `gin` key does not remove a span. The gin instrumentation does not start a span. It only renames the `net/http` server span to `METHOD /route`, sets `http.route`, and records gin errors on that span.

The `net/http` server hook and the gRPC hooks have no step-aside logic yet. [Issue #69](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/issues/69) tracks the open design question.

## Hook and Instrumentation Limitations

The following constraints apply to all hook implementations. They come from the compile-time injection model and cannot be worked around.

**Restricted imports.** Hook code is injected into the target library's compile unit, so it must not introduce circular dependencies. Imports from the target library itself, OpenTelemetry packages, and the Go standard library are always safe. Additional third-party packages are possible if declared in the instrumentation module's `go.mod`, but keep them to a minimum — every extra dependency widens the transitive dependency graph for every user of the instrumentation. See [Adding a New Instrumentation Hook](instrument-guide.md#limitations).

**Generic functions.** When the target function is generic, `HookContext` APIs that modify parameters or return values (`SetParam`, `SetReturnVal`) cannot be used. The type parameters are not available to the injected code.

**Package initialization order.** A hook can run before its own package's variable initializers and `init()` functions, when the instrumented function is called during package initialization (for example from a driver's `init()`). The hook package is only imported from package `main`, so Go does not order it before the target package. A hook that uses state set up at initialization time panics, and the trampoline prints `failed to exec Before hook` or `failed to exec After hook` while the build and the tests still pass. Hooks must create such state on first use. See [Adding a New Instrumentation Hook](instrument-guide.md#limitations).

Operations that run during package initialization, before `pkg/runtime` has initialized, are skipped by the instrumentation and emit no telemetry. This covers, for example, a database query or an HTTP request made from an `init()` function or a package-level variable initializer. Clients and servers created at that time are still instrumented, so the calls they make after initialization are traced.

**Goroutine-local storage scope.** Some instrumentation uses goroutine-local storage (GLS) to propagate trace context across boundaries that do not pass `context.Context`. GLS state is scoped to the goroutine that wrote it: spans started in a goroutine spawned from the instrumented function are not automatically visible to that function's GLS slot. GLS is an internal mechanism, not a public API. See [GLS operation notes](instrument-guide.md#gls-operation-for-otel-sdk-instrumentation) for the full operational model.

## Getting Help

- [GitHub Discussions](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/discussions) — ask questions and discuss usage.
- [GitHub Issues](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/issues) — report bugs or request features.
- [Slack Channel](https://cloud-native.slack.com/archives/C088D8GSSSF) — `#otel-go-compile-instrumentation` channel.
