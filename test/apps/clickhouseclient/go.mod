module go.opentelemetry.io/otelc-contrib/test/apps/clickhouseclient

go 1.26.0

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.50.0
	go.opentelemetry.io/otelc-contrib/instrumentation/database/sql/otelcsql v0.0.0-20261119084005-f5630b90104d
	go.opentelemetry.io/otelc-contrib/instrumentation/github.com/ClickHouse/clickhouse-go/v2/otelcclickhouse v0.0.0
	go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init v0.0.0-20261119084005-f5630b90104d
)

replace (
	go.opentelemetry.io/otelc-contrib/instrumentation/database/sql/otelcsql => ../../../instrumentation/database/sql/otelcsql
	go.opentelemetry.io/otelc-contrib/instrumentation/github.com/ClickHouse/clickhouse-go/v2/otelcclickhouse => ../../../instrumentation/github.com/ClickHouse/clickhouse-go/v2/otelcclickhouse
	go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/init => ../../../instrumentation/go.opentelemetry.io/otel/init
)

require (
	github.com/ClickHouse/ch-go v0.74.0 // indirect
	github.com/andybalholm/brotli v1.2.6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.31.0 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/paulmach/orb v0.13.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.72.0 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.5.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/prometheus v0.72.0 // indirect
	go.opentelemetry.io/contrib/exporters/autoexport v0.72.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.72.0 // indirect
	go.opentelemetry.io/contrib/propagators/autoprop v0.72.0 // indirect
	go.opentelemetry.io/contrib/propagators/aws v1.47.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.47.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.47.0 // indirect
	go.opentelemetry.io/contrib/propagators/ot v1.47.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.69.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/log v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace v0.0.0-20261119084005-f5630b90104d // indirect
	go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/trace/otelctrace v0.0.0-20261119084005-f5630b90104d // indirect
	go.opentelemetry.io/otelc-contrib/pkg/runtime v0.0.0-20261119084005-f5630b90104d // indirect
	go.opentelemetry.io/otelc/pkg v0.1.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/grpc v1.85.0-dev.0.20260825072537-93e31b48545e // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace go.opentelemetry.io/otelc-contrib/pkg/runtime => ../../../pkg/runtime

replace go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/trace/otelctrace => ../../../instrumentation/go.opentelemetry.io/otel/trace/otelctrace

replace go.opentelemetry.io/otelc-contrib/instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace => ../../../instrumentation/go.opentelemetry.io/otel/sdk/trace/otelctrace
