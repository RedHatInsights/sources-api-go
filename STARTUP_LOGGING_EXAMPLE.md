# Startup Logging Example

This document shows the expected log output during application startup after the logging improvements.

## Example Startup Log Sequence (API Server Mode)

```json
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting Sources API application...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing encryption and logger (log level: info)", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Redis cache...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to Redis/Valkey at redis-host:6379...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Redis/Valkey connection established at redis-host:6379", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing database connection and running migrations...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to database at db-host:5432 (database: sources)...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Database connection established", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Running database migrations...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Database migrations completed successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing secret store (type: vault)...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Vault secret store initialized", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Seeding database...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Database seeding completed", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Populating static type cache...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Static type cache populated successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Kafka producer for topic: platform.sources.superkey-requests...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Kafka producer initialized successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Prometheus metrics service...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Prometheus metrics service initialized", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting metrics exporter...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting application in API Server mode...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Metrics Server started on :9000", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Application initialization completed - ready to serve requests", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"API Server started on :8000", "app":"sources-api-go", ...}
```

## Example Startup Log Sequence (Status Listener Mode)

```json
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting Sources API application...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing encryption and logger (log level: info)", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Redis cache...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to Redis/Valkey at redis-host:6379...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Redis/Valkey connection established at redis-host:6379", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing database connection and running migrations...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to database at db-host:5432 (database: sources)...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Database connection established", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Running database migrations...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Database migrations completed successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing secret store (type: vault)...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Vault secret store initialized", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Populating static type cache...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Static type cache populated successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Kafka producer for topic: platform.sources.superkey-requests...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Kafka producer initialized successfully", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Initializing Prometheus metrics service...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Prometheus metrics service initialized", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting metrics exporter...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Starting application in Status Listener mode...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Metrics Server started on :9000", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Application initialization completed - ready to serve requests", "app":"sources-api-go", ...}
```

## Error Scenarios

### Database Connection Failure
```json
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to database at db-host:5432 (database: sources)...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"error", "message":"Failed to connect to database at db-host:5432: dial tcp db-host:5432: connect: connection refused", "app":"sources-api-go", ...}
```

### Redis Connection Failure
```json
{"@timestamp":"2026-07-07T...", "level":"info", "message":"Connecting to Redis/Valkey at redis-host:6379...", "app":"sources-api-go", ...}
{"@timestamp":"2026-07-07T...", "level":"error", "message":"Failed to connect to Redis/Valkey at redis-host:6379: dial tcp redis-host:6379: connect: connection refused", "app":"sources-api-go", ...}
```

## Key Improvements

1. **Clear startup indicator**: First log message clearly states the application is starting
2. **Database visibility**: Shows database hostname, port, and database name during connection attempt
3. **Connection status**: Explicit success/failure messages for both Redis and database connections
4. **Migration tracking**: Clear indication when migrations start and complete
5. **Service initialization**: Each major service (Kafka, Metrics, Secret Store) logs its initialization
6. **Application mode**: Clear indication of which mode the application is running in (API Server, Status Listener, Background Worker)
7. **Ready state**: Final message indicates all initialization is complete and the application is ready
8. **Error context**: Error messages include relevant connection details (hostname, port) for faster troubleshooting

## Log Format

All logs are output in JSON format compatible with standard log aggregation tools:
- Kibana (via CloudWatch)
- Grafana
- Zinc (development environments)

The `@timestamp`, `level`, `message`, and `app` fields are consistently included in all log entries.
