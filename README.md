<div align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-4A90E2?style=flat&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/License-Apache%202.0-4A90E2?style=flat&logo=apache" alt="License" />
  <img src="https://img.shields.io/badge/Status-Experimental-4A90E2?style=flat&logo=github" alt="Status" />
  <img src="https://img.shields.io/badge/Slack-CNCF-FF6B35?style=flat&logo=slack" alt="Slack" />
</div>

# OpenTelemetry Go Compile Contrib

This repository hosts community-contributed compile-time instrumentations and supporting runtime packages for
[`opentelemetry-go-compile-instrumentation`][otelc] (`otelc`), OpenTelemetry's compile-time instrumentation
toolchain for Go.

Separating instrumentations from the core `otelc` tool allows new libraries to be instrumented,
tested, and released independently of the compiler toolchain, and enables manifest- and registry-based discovery.

[otelc]: https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation

## Supported Instrumentations

The table below lists all currently supported instrumentation packages. All packages require `otelc >= v1.2.0`.

| Category                  | Target Library                                   | Module Path                                                                                                                                                                                                                                                                                     | Telemetry Emitted                              |
| :------------------------ | :----------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------------- |
| **HTTP / Web**            | `net/http`                                       | `go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp`                                                                                                                                                                                                                          | Client & server spans, HTTP metrics            |
|                           | `github.com/gin-gonic/gin`                       | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/gin-gonic/gin/otelcgin`                                                                                                                                                                                                           | Server span route enrichment & errors          |
|                           | `github.com/gofiber/fiber/v2`                    | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/gofiber/fiber/v2/otelcfiber`                                                                                                                                                                                                      | Server spans, status code, route               |
|                           | `github.com/labstack/echo/v4`                    | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/labstack/echo/v4/otelcecho`                                                                                                                                                                                                       | Server span route enrichment & errors          |
| **RPC & Cloud**           | `google.golang.org/grpc`                         | `go.opentelemetry.io/otelc-contrib/instrumentation/google.golang.org/grpc/otelcgrpc`                                                                                                                                                                                                            | Client & server RPC spans and metrics          |
|                           | `github.com/aws/aws-sdk-go-v2`                   | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelcaws`                                                                                                                                                                                                       | AWS SDK client operation spans                 |
|                           | `k8s.io/client-go`                               | `go.opentelemetry.io/otelc-contrib/instrumentation/k8s.io/client-go/otelcclientgo`                                                                                                                                                                                                              | Informer event handler spans                   |
|                           | `github.com/linode/linodego/v2`                  | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/linode/linodego/v2/otelclinodego`                                                                                                                                                                                                 | HTTP client spans, duration metric             |
|                           | `github.com/stripe/stripe-go` (v81, v82)         | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/stripe/stripe-go/v81/otelcstripe`<br>`go.opentelemetry.io/otelc-contrib/instrumentation/github.com/stripe/stripe-go/v82/otelcstripe`                                                                                              | Client request spans, duration metric          |
| **Databases & Caches**    | `database/sql`                                   | `go.opentelemetry.io/otelc-contrib/instrumentation/database/sql/otelcsql`                                                                                                                                                                                                                       | DB client spans (queries, execs, txs)          |
|                           | `github.com/redis/go-redis/v9`                   | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/redis/go-redis/v9/otelcredis`                                                                                                                                                                                                     | Redis client command & pipeline spans          |
|                           | `go.mongodb.org/mongo-driver` (v1, v2)           | `go.opentelemetry.io/otelc-contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelcmongo`<br>`go.opentelemetry.io/otelc-contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelcmongo`                                                                                         | MongoDB command spans                          |
|                           | `github.com/olivere/elastic/v7`                  | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/olivere/elastic/v7/otelcelastic`                                                                                                                                                                                                  | Elasticsearch client request spans             |
|                           | `github.com/opensearch-project/opensearch-go/v5` | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/opensearch-project/opensearch-go/v5/otelcopensearch`                                                                                                                                                                              | OpenSearch client request spans                |
|                           | `github.com/apache/cassandra-gocql-driver/v2`    | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/apache/cassandra-gocql-driver/v2/otelcgocql`                                                                                                                                                                                      | Cassandra CQL query spans                      |
| **Messaging**             | `github.com/segmentio/kafka-go`                  | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/segmentio/kafka-go/otelckafka`                                                                                                                                                                                                    | Kafka producer & consumer message spans        |
|                           | `github.com/rabbitmq/amqp091-go`                 | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/rabbitmq/amqp091-go/otelcamqp091`                                                                                                                                                                                                 | RabbitMQ publish & consume spans               |
| **GenAI**                 | `github.com/openai/openai-go` (v1, v2, v3)       | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/openai/openai-go/otelcopenai`<br>`go.opentelemetry.io/otelc-contrib/instrumentation/github.com/openai/openai-go/v2/otelcopenai`<br>`go.opentelemetry.io/otelc-contrib/instrumentation/github.com/openai/openai-go/v3/otelcopenai` | Client spans, tokens, streaming                |
|                           | `github.com/anthropics/anthropic-sdk-go`         | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/anthropics/anthropic-sdk-go/otelcanthropic`                                                                                                                                                                                       | Client message spans, tokens                   |
|                           | `google.golang.org/genai`                        | `go.opentelemetry.io/otelc-contrib/instrumentation/google.golang.org/genai/otelcgenai`                                                                                                                                                                                                          | Gemini client spans, tokens, streaming         |
| **Logging (Correlation)** | `log`                                            | `go.opentelemetry.io/otelc-contrib/instrumentation/log/otelclog`                                                                                                                                                                                                                                | Context injection into stdlib log              |
|                           | `log/slog`                                       | `go.opentelemetry.io/otelc-contrib/instrumentation/log/slog/otelcslog`                                                                                                                                                                                                                          | `trace_id` / `span_id` in slog handler         |
|                           | `github.com/sirupsen/logrus`                     | `go.opentelemetry.io/otelc-contrib/instrumentation/github.com/sirupsen/logrus/otelclogrus`                                                                                                                                                                                                      | `trace_id` / `span_id` fields in logrus        |
|                           | `go.uber.org/zap`                                | `go.opentelemetry.io/otelc-contrib/instrumentation/go.uber.org/zap/otelczap`                                                                                                                                                                                                                    | `trace_id` / `span_id` fields in zap           |
| **SDK & Runtime**         | `go.opentelemetry.io/otel/init`                  | `go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init`                                                                                                                                                                                                               | Automatic SDK initialization & runtime metrics |
|                           | `go.opentelemetry.io/otel`                       | `go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/otelcotel`                                                                                                                                                                                                          | Global tracer provider protection              |
|                           | `go.opentelemetry.io/otel/sdk/trace`             | `go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace`                                                                                                                                                                                               | Goroutine-local storage (GLS) span tracking    |
|                           | `go.opentelemetry.io/otel/trace`                 | `go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/trace/otelctrace`                                                                                                                                                                                                   | GLS fallback for `SpanFromContext`             |

> [!NOTE]
> Some instrumentations provide separate subpackages for client and server capabilities. For example, `net/http` (`otelchttp`) provides separate `client` and `server` packages (`.../otelchttp/client` and `.../otelchttp/server`), and `google.golang.org/grpc` (`otelcgrpc`) provides `otelcgrpc/client` and `otelcgrpc/server`. Import the specific subpackage(s) matching your application's needs.

For detailed telemetry specifications, see [`schemas/otelc-contrib/README.md`](schemas/otelc-contrib/README.md).

---

## Usage

### 1. Declare Instrumentation in Your Application

In your application's root directory, create an `otel.instrumentation.go` file (using the standard `//go:build tools` constraint) with blank imports for the desired instrumentation modules:

```go
//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp/client"
	_ "go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp/server"
)
```

> **Note**: For instrumentations with separate client and server subpackages (such as `net/http/otelchttp` and `google.golang.org/grpc/otelcgrpc`), import only the subpackage(s) required by your application.

Run `go mod tidy` to resolve dependencies:

```sh
go mod tidy
```

Alternatively, use `otelc pin` to automatically discover and add matching instrumentations based on your application's dependencies:

```sh
otelc pin
```

### 2. Build Your Application

Build using the `otelc` compiler wrapper:

```sh
otelc go build -o myapp .
```

`otelc` automatically detects the declared rules, intercepts package compilation, and weaves the hooks directly into the resulting binary.

---

## Documentation

- [Adding a New Instrumentation Hook](docs/instrument-guide.md): Guide for developing, testing, and contributing new instrumentation modules.
- [Rule Specification Reference][upstream-rules]: Canonical specification for `otelc.yaml` rule syntax, selectors, and modifiers in the upstream `otelc` repository.
- [Troubleshooting Guide](docs/troubleshooting.md): Common runtime issues, package initialization order (`runtime.Ready()`), and runtime selection gates.
- [GenAI Compatibility Matrix](docs/genai-compatibility-matrix.md): Supported features, token accounting, streaming, and semantic convention alignments for GenAI SDKs.
- [Semantic Conventions & Schema Registry](docs/semantic-conventions.md): Weaver schema registry workflow, code generation (`make generate`), and validation (`make lint-schema`).
- [Testing Strategy](docs/testing.md): Unit tests, integration tests against live apps, E2E tests, and upstream compatibility matrices (`latestlibbuild`, `latestlibrun`, `versionmatrix`).

[upstream-rules]: https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation/blob/main/docs/rules.md

---

## Root span filtering

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

## Community

### Get Help

- [GitHub Discussions](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/discussions) - Ask questions
- [GitHub Issues](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/issues) - Report bugs
- [Slack Channel](https://cloud-native.slack.com/archives/C088D8GSSSF) - Real-time chat
- [Calendar](https://github.com/open-telemetry/community/#sig-go-compile-instrumentation) - Community meetings (Thursdays, UTC: 09:30 – 10:30)

## Contributing

We welcome contributions! See our [contributing guide](CONTRIBUTING.md) and [development docs](./docs/).

This project follows the [OpenTelemetry Code of Conduct](https://github.com/open-telemetry/community/blob/main/code-of-conduct.md).
Please also review our [AI usage policy](docs/AI_POLICY.md) if you use AI tools in your workflow.

Here is a list of community roles with current and previous members:

### Maintainers

- [Azhar Momin](https://github.com/amazingakai), Independent
- [Dario Castañe](https://github.com/darccio), Datadog
- [Haibin Zhang](https://github.com/NameHaibinZhang), Alibaba
- [Huxing Zhang](https://github.com/ralf0131), Alibaba
- [Kemal Akkoyun](https://github.com/kakkoyun), Datadog
- [Przemyslaw Delewski](https://github.com/pdelewski), Quesma
- [Xabier Martinez](https://github.com/txabman42), Cabify
- [Yi Yang](https://github.com/y1yang0), Alibaba

For more information about the maintainer role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#maintainer).

### Approvers

For more information about the approver role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#approver).

### Thanks to all of our contributors!

<a href="https://github.com/open-telemetry/opentelemetry-go-compile-contrib/graphs/contributors">
  <img alt="Repo contributors" src="https://contrib.rocks/image?repo=open-telemetry/opentelemetry-go-compile-contrib" />
</a>
