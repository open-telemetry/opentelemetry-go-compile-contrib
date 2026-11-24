# Plan: PR 2 — Documentation Overhaul & v0.1.0 Polish

## Objective
Update and polish all documentation across `opentelemetry-go-compile-contrib` in preparation for the `v0.1.0` release. Establish a clean, permanent documentation boundary between the core compiler repository (`otelc`) and this repository (`otelc-contrib`), remove all migration-era banners, clean out stale compiler-internal references, update import paths and directory structures to contrib conventions, bring over essential upstream documentation needed by instrumentation users, and provide a developer-ready guide for contributors and users.

---

## 1. Architectural & Documentation Boundary

The split between `open-telemetry/opentelemetry-go-compile-instrumentation` (Core `otelc`) and `open-telemetry/opentelemetry-go-compile-contrib` (Contrib) establishes a clear separation of concerns:

```
┌────────────────────────────────────────────────────────┐
│  Core: open-telemetry/opentelemetry-go-compile-        │
│        instrumentation (Toolchain & Engine)            │
├────────────────────────────────────────────────────────┤
│ • Compiler binary: otelc go build, otelc pin           │
│ • Build interception: -toolexec, AST rewriting         │
│ • Canonical Rule Engine Specification (docs/rules.md)  │
│ • Public Hook API contract: pkg/hook (HookContext)     │
│ • Manifest consumer: reads instrumentation-manifest    │
│ • Compiler integration tests with synthetic fixtures   │
│ • Architecture decision records: docs/adr/             │
└────────────────────────────────────────────────────────┘
                           ▲
                           │ imports pkg/hook & references rules.md
                           │
┌────────────────────────────────────────────────────────┐
│  Contrib: open-telemetry/opentelemetry-go-compile-     │
│           contrib (Instrumentation Ecosystem)          │
├────────────────────────────────────────────────────────┤
│ • All hook modules: instrumentation/** (30+ packages)  │
│ • Supporting runtime packages: pkg/runtime, pkg/netutil│
│ • Generated semconv constants: pkg/otelcconv           │
│ • Weaver schema registry: schemas/otelc-contrib/       │
│ • Integration & E2E tests: test/apps/, test/e2e/       │
│ • Upstream compatibility: latestlib*, versionmatrix    │
│ • Manifest producer: generates catalog metadata        │
└────────────────────────────────────────────────────────┘
```

### What is Kept in Core (`otelc`)
The core repository documents the compiler toolchain, its CLI, and compiler-level features:
1. **`README.md`**: Focuses on `otelc` installation (`go install go.opentelemetry.io/otelc/tool/cmd/otelc@latest`), CLI commands (`otelc go build`, `otelc pin`), and prominently directs users to `otelc-contrib` for all available instrumentations.
2. **`docs/getting-started.md`**: Getting started with `otelc` CLI and project setup.
3. **`docs/configuration.md`**: Compiler flags (`-log-level`, `-build-plan`, `--timeout`), environment variables, and manifest discovery.
4. **`docs/external-configuration.md`**: How `otelc` scans and parses `otel.instrumentation.go` files and `//go:build tools`.
5. **`docs/troubleshooting.md`**: Compiler build errors, build cache issues (`GOCACHE`), debug logs (`matched.json`, `debug.log`), link-time batching issues. Points to `otelc-contrib` for instrumentation troubleshooting.
6. **`docs/benchmarking.md` & `docs/profiling.md`**: Compiler build overhead benchmarking and CPU/memory profiling.
7. **`docs/api-design-and-project-structure.md`**: Compiler architecture, AST rewriting engine, link-time trampoline generation, and the `pkg/hook` API contract.
8. **`docs/rules.md`**: Canonical rule engine specification implemented by the compiler. (Contrib links to this upstream document rather than duplicating it).
9. **`docs/testing.md`**: Compiler integration tests with synthetic fixtures and compiler unit tests.
10. **`docs/adr/`**: All Architectural Decision Records (ADRs stay in core).
11. **`docs/RELEASE.md`**: Release processes for the compiler binary and `pkg/hook`.

### What Will Stay / Live in Contrib (`otelc-contrib`)
The contrib repository documents the instrumentation ecosystem, telemetry contracts, and contribution workflows:
1. **`README.md`**: Supported Instrumentations Catalog (table of 24+ supported libraries), compatibility with `otelc` (`>= v1.2.0`), release status (`v0.1.0`), and usage instructions with `otel.instrumentation.go`.
2. **`docs/instrument-guide.md`**: Guide for writing new instrumentation hooks, runtime gate (`Enable()`), `runtime.Ready()` initialization check, hook signatures (`hook.HookContext`), testing, and directory layout (`instrumentation/<import_path>/otelc<pkg>/`). Links to core's `rules.md` for rule syntax.
3. **`docs/semantic-conventions.md`**: Weaver semantic conventions workflow, `schemas/otelc-contrib/` registry contract, code generation to `pkg/otelcconv/`, `make lint-schema`, `make generate`.
4. **`docs/genai-compatibility-matrix.md`** (*migrated from `otelc`*): Supported GenAI SDKs (`openai-go` v1/v2/v3, `anthropic-sdk-go`, `google.golang.org/genai`), streaming support, token accounting, and registry alignment.
5. **`docs/troubleshooting.md`** (*new in contrib*): Troubleshooting for instrumentation consumers:
   - Package initialization order & `runtime.Ready()` (from PR #1717 / commit `2aecaeb5`).
   - Runtime enable/disable flags (`OTEL_GO_ENABLED_INSTRUMENTATIONS` / `OTEL_GO_DISABLED_INSTRUMENTATIONS`).
   - GLS scope and fallback.
6. **`docs/testing.md`**: Contrib test strategy: unit tests (`instrumentation/**`, `pkg/**`), integration tests (`test/integration/**`), E2E tests (`test/e2e/**`), and compatibility matrix tests (`latestlibbuild`, `latestlibrun`, `versionmatrix`).
7. **`CONTRIBUTING.md`**: Workflow for adding/modifying instrumentations, schemas, running `make all`, conventional commits, and merge criteria.
8. **`AGENTS.md`**: Guidelines for AI-assisted contributions to `otelc-contrib`.

---

## 2. File-by-File Changes for PR 2

### A. Root Documentation & Contributor Guides

#### 1. `README.md`
- **Remove Migration Language**:
  - Delete Section `## Status` ("newly created and being populated as part of migration...").
  - Remove links and references to upstream migration epic #1260.
- **Add v0.1.0 Release & Status**:
  - State the release status (`v0.1.0`) and compatibility with `otelc` (`>= v1.2.0`).
- **Supported Instrumentations Catalog**:
  - Add a comprehensive table of all supported instrumentation modules:
    - **HTTP / Web Frameworks**: `net/http` (`otelchttp`), `gin-gonic/gin` (`otelcgin`), `gofiber/fiber/v2` (`otelcfiber`), `labstack/echo/v4` (`otelcecho`).
    - **RPC & Cloud**: `google.golang.org/grpc` (`otelcgrpc`), `aws/aws-sdk-go-v2` (`otelcaws`), `k8s.io/client-go` (`otelcclientgo`), `linode/linodego/v2` (`otelclinodego`), `stripe/stripe-go` (`otelcstripe` v81, v82).
    - **Databases & Caches**: `database/sql` (`otelcsql`), `redis/go-redis/v9` (`otelcredis`), `go.mongodb.org/mongo-driver` (`otelcmongo` v1, v2), `olivere/elastic/v7` (`otelcelastic`), `opensearch-project/opensearch-go/v5` (`otelcopensearch`), `gocql/gocql` (`otelcgocql`).
    - **Messaging**: `segmentio/kafka-go` (`otelckafka`), `rabbitmq/amqp091-go` (`otelcamqp091`).
    - **GenAI**: `openai/openai-go` (`otelcopenai` v1, v2, v3), `anthropics/anthropic-sdk-go` (`otelcanthropic`), `google.golang.org/genai` (`otelcgenai`).
    - **Logging (Trace Correlation)**: `log` (`otelclog`), `log/slog` (`otelcslog`), `sirupsen/logrus` (`otelclogrus`), `uber-go/zap` (`otelczap`).
    - **SDK & Runtime**: `go.opentelemetry.io/otel/init` (`otelcinit`), `go.opentelemetry.io/otel/otelcotel`, `go.opentelemetry.io/otel/sdk/trace/otelctrace`, `go.opentelemetry.io/otel/trace/otelctrace`.
- **Usage & Selection Guide**:
  - Explain how target applications declare compile-time instrumentation dependencies via `otel.instrumentation.go` (`//go:build tools`, `package tools`).
  - Explain how `otelc` automatically discovers and links them during compilation (`otelc pin`, `otelc go build`).

#### 2. `CONTRIBUTING.md`
- **Remove Compiler-Specific References**:
  - Remove Windows/WSL golden files under `testdata/golden` (line 25-33), as AST golden file tests belong to core `otelc`.
  - Clarify the role of `make otelc`: in contrib, it downloads/installs the `otelc` compiler tool into `.bin/` from `.tools` for testing.
- **Clarify Tool vs Contrib Relationship**:
  - State clearly that `otelc` is the external compiler tool and `otelc-contrib` is the repository for instrumentation modules, schemas, and upstream test suites.
  - Detail available make targets and guidelines for adding new instrumentations.

#### 3. `AGENTS.md`
- Update document links to include `docs/instrument-guide.md` and link upstream `docs/rules.md`.
- Reiterate that all instrumentation development, semconv changes, and upstream testing occur in `otelc-contrib`.

---

### B. Core Developer Guides

#### 4. `docs/instrument-guide.md`
- **Path & Module Alignment**:
  - Update all example module and import paths from `go.opentelemetry.io/otelc/...` to `go.opentelemetry.io/otelc-contrib/...`.
  - Update directory layout examples to reflect the contrib `otelc<pkg>` directory standard (e.g. `instrumentation/google.golang.org/grpc/otelcgrpc/server/`).
- **Package Initialization Order & `runtime.Ready()` (from upstream PR #1717)**:
  - Document that hooks running during package initialization must check `runtime.Ready()` before starting telemetry to avoid panics on uninitialized OpenTelemetry globals.
- **Link Upstream Rule Specification**:
  - Replace relative `rules.md` links with link to upstream `open-telemetry/opentelemetry-go-compile-instrumentation`'s `docs/rules.md`.
  - Link to `docs/troubleshooting.md` for runtime symptoms.
- **Update GLS Section**:
  - Update file references from `instrumentation/go.opentelemetry.io/otel/sdk/trace/otel_trace_context.go` to `instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace/otel_trace_context.go`.
  - Clarify that low-level runtime GLS injection is provided by the `otelc` compiler tool, while trace state management is implemented in `otelctrace`.

#### 5. `docs/semantic-conventions.md`
- **Remove Migration Callout**:
  - Delete `> **Migration in Progress**` banner.
- **Document Code Generation (`pkg/otelcconv`)**:
  - Document the code generation workflow: how `schemas/otelc-contrib/` local attributes with `id:` generate Go constants into `pkg/otelcconv/` using `make generate` and OTel Weaver.
  - Update Section "Workflow: Adding a New Attribute" to use `make generate` rather than manual constant authoring.
- **Clean Stale Targets**:
  - Remove `make build-demo` reference (line 403).

#### 6. `docs/testing.md`
- **Remove Migration Callout**:
  - Delete `> **Migration in Progress**` banner.
- **Clean Compiler-Internal References**:
  - Remove references to `tool/` module tests, `make test-unit/tool`, and golden file tests (`tool/internal/instrument/testdata/golden`, `make test-unit/update-golden`).
  - Remove references to `demo/app/basic/instrumentation` and `make build-demo`.
  - Remove tool coverage floor statements (`≥68% for tool/ module tree`).
  - Remove references to dropped stdlib `runtime` field injection.
- **Document Contrib Test Architecture**:
  - **Unit Tests**: `make test-unit` (`instrumentation/**`, `pkg/**`, `test/testutil/**`).
  - **Integration Tests**: `make test-integration` (executing `test/apps/**` built with `otelc go build` and verifying exported OTLP spans and metrics).
  - **E2E Tests**: `make test-e2e` (multi-process tests such as Mongo, gRPC, and HTTP client/server).
  - **Regression & Matrix Tests**: `make test-latestlibbuild`, `make test-latestlibrun`, `make test-versionmatrix`.
  - Document the wiring pattern: how test apps use `otel.instrumentation.go` to declare contrib modules.

---

### C. Upstream Documentation Migrated to Contrib

#### 7. `docs/genai-compatibility-matrix.md` (Migrated from Core)
- Adopt `docs/genai-compatibility-matrix.md` (dropped from core in PR 2).
- Update status for `google.golang.org/genai` (`otelcgenai`): promote from "Planned targets" to "Currently instrumented" (reflecting the merge of PR #1153 and streaming support in PR #1829).
- Document supported versions, token usage accounting, streaming support, and registry group alignments for OpenAI, Anthropic, and Gemini.

#### 8. `docs/troubleshooting.md` (New in Contrib)
- Create a dedicated troubleshooting guide for instrumentation users and authors:
  - **Package Initialization Order**: Guidance on `runtime.Ready()` and why operations during `init()` are skipped (from PR #1717).
  - **Runtime Enable/Disable**: Debugging `OTEL_GO_ENABLED_INSTRUMENTATIONS` and `OTEL_GO_DISABLED_INSTRUMENTATIONS`.
  - **GLS Scope & Fallback**: How goroutine-local storage works across goroutine boundaries.

---

### D. Stale References in Instrumentation READMEs
- In `instrumentation/github.com/linode/linodego/v2/otelclinodego/README.md`: update import path from `go.opentelemetry.io/otelc/...` to `go.opentelemetry.io/otelc-contrib/...`.
- In `instrumentation/github.com/stripe/stripe-go/v81/otelcstripe/README.md`: update import path to `otelc-contrib`.
- In `instrumentation/github.com/stripe/stripe-go/v82/otelcstripe/README.md`: update import path to `otelc-contrib`.
- In `instrumentation/net/http/otelchttp/README.md`: fix example rule paths to include `otelchttp`.

---

## 3. Verification & Quality Gates

Before finalizing PR 2:
1. `make format`: Ensure all Go and YAML files are formatted.
2. `make lint`: Run all linters (Go, YAML, Actions, Makefile, License).
3. Validate all markdown links: ensure no broken internal or cross-repo links.
4. Verify that no references to `go.opentelemetry.io/otelc/instrumentation` remain in contrib documentation.
