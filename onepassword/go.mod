module dagger/onepassword

go 1.23.1

require github.com/1password/onepassword-sdk-go v0.1.1

require (
	github.com/extism/go-sdk v1.3.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/tetratelabs/wazero v1.7.3 // indirect
)

replace go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc => go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0

replace go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp => go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0

replace go.opentelemetry.io/otel/log => go.opentelemetry.io/otel/log v0.19.0

replace go.opentelemetry.io/otel/sdk/log => go.opentelemetry.io/otel/sdk/log v0.19.0
