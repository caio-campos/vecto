# Como Rodar os Exemplos

## Pré-requisitos

- Go 1.25 ou superior
- Conexão com internet (os exemplos fazem requisições HTTP)

## Executando os Exemplos

Cada exemplo usa build tags e deve ser executado individualmente:

### Enhanced Request
```bash
go run -tags example_enhanced_request examples/enhanced_request.go
```

### Auth
```bash
go run -tags example_auth examples/auth.go
```

### Retry
```bash
go run -tags example_retry examples/retry.go
```

### Circuit Breaker
```bash
go run -tags example_circuit_breaker examples/circuit_breaker.go
```

### Custom Logger
```bash
go run -tags example_custom_logger examples/custom_logger.go
```

### Metrics Collector
```bash
go run -tags example_metrics_collector examples/metrics_collector.go
```

### Trace Debug
```bash
go run -tags example_trace_debug examples/trace_debug.go
```

## Executando a partir da pasta examples

Se estiver dentro da pasta `examples`:

```bash
cd examples
go run -tags example_enhanced_request enhanced_request.go
```

