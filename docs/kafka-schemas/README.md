# Kafka Message Schemas for Availability Checks

**Version:** 1.0  
**Status:** Draft  
**Date:** 2026-10-07  
**JIRA:** RHCLOUD-52010

---

## Overview

This document defines the Kafka message schemas for event-driven availability checking in Sources. These schemas define the contract between Sources (producer/consumer) and application services (Cost Management, cloud-meter, Provisioning, Cloud Connector).

**Note:** This replaces the previous HTTP-based availability check mechanism. For migration context, see [Architecture Evaluation Document](../../../availability-check-architecture-evaluation.md).

---

## Schema Format

**Format:** JSON Schema (Draft 2020-12)  
**Encoding:** JSON (UTF-8)  
**Content-Type:** `application/json`

**Why JSON Schema?**
- Human-readable and easy to implement across languages (Go, Python, etc.)
- Widely supported validation libraries
- Simpler than Avro for this use case
- No schema registry infrastructure required initially

**Future Considerations:**
- If message volume becomes extremely high (>100k/day), consider Avro for better compression
- Schema registry (e.g., Confluent Schema Registry) can be added later for schema versioning

---

## Topics

### Topic 1: Availability Check Requests

**Name:** `platform.sources.availability-check-requests`  
**Purpose:** Sources publishes requests for applications to perform availability checks  
**Producer:** `sources-api-go`  
**Consumers:** Application services (Cost Management, cloud-meter, Provisioning, Cloud Connector)

**Configuration:**
- Partitions: 3 (scalable for multi-tenant load)
- Replication factor: 3 (production-ready)
- Retention: 7 days (sufficient for troubleshooting)
- Partition key: `source_id` (ensures ordering per source)
- Cleanup policy: `delete`

### Topic 2: Availability Check Results

**Name:** `platform.sources.availability-check-results`  
**Purpose:** Applications publish results of availability checks back to Sources  
**Producers:** Application services (Cost Management, cloud-meter, Provisioning, Cloud Connector)  
**Consumer:** `sources-api-go`

**Configuration:**
- Partitions: 3
- Replication factor: 3
- Retention: 30 days (longer for audit trail and debugging)
- Partition key: `source_id`
- Cleanup policy: `delete`

---

## Message Schemas

### 1. Availability Check Request

**Topic:** `platform.sources.availability-check-requests`  
**Schema File:** [`availability-check-request.schema.json`](./availability-check-request.schema.json)

#### Message Structure

```json
{
  "source_id": "12345",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "application_id": "67890",
  "application_type": "/insights/platform/cost-management",
  "source_type": "amazon",
  "source_name": "AWS Production Account",
  "timestamp": "2026-10-07T14:30:00.000Z",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440001",
  "request_metadata": {
    "triggered_by": "manual|scheduled|api",
    "user_id": "user@redhat.com"
  }
}
```

#### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source_id` | string | ✅ Yes | Unique identifier for the source in Sources database |
| `tenant_id` | string (UUID) | ✅ Yes | Red Hat account/organization ID (org_id) |
| `application_id` | string | ✅ Yes | Unique identifier for the application instance |
| `application_type` | string | ✅ Yes | Application type path (e.g., `/insights/platform/cost-management`) |
| `source_type` | string | ✅ Yes | Type of external system (`amazon`, `azure`, `google`, `openshift`, `ibm`, `github`, etc.) |
| `source_name` | string | ❌ No | Human-readable source name (for logging/debugging) |
| `timestamp` | string (ISO 8601) | ✅ Yes | UTC timestamp when request was published |
| `correlation_id` | string (UUID) | ✅ Yes | UUID v4 for request/result correlation |
| `request_metadata` | object | ❌ No | Additional context about the request |
| `request_metadata.triggered_by` | enum | ❌ No | How check was triggered: `manual`, `scheduled`, `api` |
| `request_metadata.user_id` | string | ❌ No | User who triggered manual check (if applicable) |

#### Kafka Message Properties

- **Key:** `source_id` (string) — Ensures all events for the same source go to the same partition (ordering guarantee)
- **Headers:**
  - `event_type`: `"availability-check-request"` (for consumer filtering)
  - `schema_version`: `"1.0"` (for schema evolution)
  - `tenant_id`: `<tenant_id>` (for multi-tenancy filtering)

#### Consumer Filtering

Applications should filter messages by `application_type`:

```python
# Python example (Cost Management)
if message.value['application_type'] == '/insights/platform/cost-management':
    process_availability_check(message.value)
```

```go
// Go example (Cloud Connector)
if msg.Value.ApplicationType == "cloud-connector" {
    processAvailabilityCheck(msg.Value)
}
```

#### Example Use Cases

**Use Case 1: Scheduled Check (sources-monitor-go)**
```json
{
  "source_id": "12345",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "application_id": "67890",
  "application_type": "/insights/platform/cost-management",
  "source_type": "amazon",
  "source_name": "AWS Production",
  "timestamp": "2026-10-07T12:00:00.000Z",
  "correlation_id": "c7f8e4a0-8b3c-4f1e-9d2a-5e6f7a8b9c0d",
  "request_metadata": {
    "triggered_by": "scheduled"
  }
}
```

**Use Case 2: Manual Check (UI Button)**
```json
{
  "source_id": "12345",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "application_id": "67890",
  "application_type": "/insights/platform/cloud-meter",
  "source_type": "azure",
  "source_name": "Azure Test Subscription",
  "timestamp": "2026-10-07T14:30:45.123Z",
  "correlation_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
  "request_metadata": {
    "triggered_by": "manual",
    "user_id": "user@redhat.com"
  }
}
```

**Use Case 3: RHC Connection Check (Cloud Connector)**
```json
{
  "source_id": "12345",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "application_id": "67890",
  "application_type": "cloud-connector",
  "source_type": "satellite",
  "source_name": "On-Premise Satellite",
  "timestamp": "2026-10-07T12:15:00.000Z",
  "correlation_id": "f8e7d6c5-b4a3-2190-8765-4321dcba9876",
  "request_metadata": {
    "triggered_by": "scheduled"
  }
}
```

---

### 2. Availability Check Result

**Topic:** `platform.sources.availability-check-results`  
**Schema File:** [`availability-check-result.schema.json`](./availability-check-result.schema.json)

#### Message Structure

```json
{
  "source_id": "12345",
  "application_id": "67890",
  "status": "available",
  "error": null,
  "error_code": null,
  "checked_at": "2026-10-07T14:30:05.123Z",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440001",
  "check_duration_ms": 3245,
  "result_metadata": {
    "cloud_provider_response_code": 200,
    "validation_details": "Successfully validated IAM role permissions"
  }
}
```

#### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source_id` | string | ✅ Yes | Unique identifier for the source (must match request) |
| `application_id` | string | ✅ Yes | Application instance ID (must match request) |
| `status` | enum | ✅ Yes | Result status: `available`, `unavailable`, `in_progress` |
| `error` | string \| null | ❌ No | Human-readable error message (null if status=available) |
| `error_code` | string \| null | ❌ No | Machine-readable error code for categorization |
| `checked_at` | string (ISO 8601) | ✅ Yes | UTC timestamp when check was completed |
| `correlation_id` | string (UUID) | ✅ Yes | UUID from request for correlation |
| `check_duration_ms` | integer | ❌ No | Duration of check execution in milliseconds (for performance monitoring) |
| `result_metadata` | object | ❌ No | Additional check-specific details |
| `result_metadata.cloud_provider_response_code` | integer | ❌ No | HTTP status code from cloud provider API (if applicable) |
| `result_metadata.validation_details` | string | ❌ No | Details about what was validated |

#### Status Values

| Status | Description | When to Use |
|--------|-------------|-------------|
| `available` | Source connection is working | Successful validation of credentials and access |
| `unavailable` | Source connection is not working | Failed validation (credentials expired, deleted, network issue, etc.) |
| `in_progress` | Check is still running | For long-running checks (rarely used, most checks complete quickly) |

**Note:** Most checks should complete quickly (<10 seconds). If a check takes longer than 15 minutes, Sources will mark it as timed out.

#### Error Codes

Standardized error codes for common failure scenarios:

| Code | Description | Example |
|------|-------------|---------|
| `AUTH_INVALID` | Credentials are invalid or expired | AWS IAM role invalid |
| `AUTH_MISSING` | No credentials found | Source has no authentication configured |
| `PERMISSION_DENIED` | Credentials lack required permissions | IAM role missing required policy |
| `RESOURCE_NOT_FOUND` | Cloud resource not found | AWS account ID not found |
| `NETWORK_ERROR` | Network connectivity issue | Cannot reach cloud provider API |
| `RATE_LIMITED` | Rate limited by cloud provider | AWS throttling |
| `TIMEOUT` | Check exceeded timeout | Cloud provider API took >10s to respond |
| `UNKNOWN` | Unknown error | Catch-all for unexpected failures |

#### Kafka Message Properties

- **Key:** `source_id` (string) — Same partition as request
- **Headers:**
  - `event_type`: `"availability-check-result"` (for consumer filtering)
  - `schema_version`: `"1.0"` (for schema evolution)
  - `correlation_id`: `<correlation_id>` (for tracing)

#### Example Results

**Example 1: Successful Check**
```json
{
  "source_id": "12345",
  "application_id": "67890",
  "status": "available",
  "error": null,
  "error_code": null,
  "checked_at": "2026-10-07T14:30:05.123Z",
  "correlation_id": "c7f8e4a0-8b3c-4f1e-9d2a-5e6f7a8b9c0d",
  "check_duration_ms": 3245,
  "result_metadata": {
    "cloud_provider_response_code": 200,
    "validation_details": "Successfully validated AWS IAM role arn:aws:iam::123456789:role/CostManagement"
  }
}
```

**Example 2: Failed Check - Invalid Credentials**
```json
{
  "source_id": "12345",
  "application_id": "67890",
  "status": "unavailable",
  "error": "AWS IAM role authentication failed: The security token included in the request is invalid",
  "error_code": "AUTH_INVALID",
  "checked_at": "2026-10-07T14:30:05.123Z",
  "correlation_id": "c7f8e4a0-8b3c-4f1e-9d2a-5e6f7a8b9c0d",
  "check_duration_ms": 1823,
  "result_metadata": {
    "cloud_provider_response_code": 403,
    "validation_details": "AWS STS AssumeRole failed"
  }
}
```

**Example 3: Failed Check - Permission Denied**
```json
{
  "source_id": "12345",
  "application_id": "67890",
  "status": "unavailable",
  "error": "IAM role lacks required permissions for Cost and Usage Report access",
  "error_code": "PERMISSION_DENIED",
  "checked_at": "2026-10-07T14:30:05.123Z",
  "correlation_id": "c7f8e4a0-8b3c-4f1e-9d2a-5e6f7a8b9c0d",
  "check_duration_ms": 2104,
  "result_metadata": {
    "cloud_provider_response_code": 403,
    "validation_details": "Missing required policy: arn:aws:iam::aws:policy/CostManagement"
  }
}
```

**Example 4: Network Error**
```json
{
  "source_id": "12345",
  "application_id": "67890",
  "status": "unavailable",
  "error": "Failed to connect to Azure API: Connection timeout after 10 seconds",
  "error_code": "NETWORK_ERROR",
  "checked_at": "2026-10-07T14:30:15.123Z",
  "correlation_id": "c7f8e4a0-8b3c-4f1e-9d2a-5e6f7a8b9c0d",
  "check_duration_ms": 10000,
  "result_metadata": {
    "validation_details": "Attempted to reach management.azure.com"
  }
}
```

---

## Correlation Strategy

### Correlation ID

**Format:** UUID v4 (RFC 4122)  
**Generated by:** Sources (when publishing request)  
**Used for:** Matching results back to requests

### Correlation Flow

```
1. Sources generates correlation_id: "a1b2c3d4-..."
2. Sources publishes request with correlation_id
3. Application receives request, extracts correlation_id
4. Application performs check
5. Application publishes result with SAME correlation_id
6. Sources matches result to original request via correlation_id
```

### Timeout Handling

If no result received within **15 minutes** of request publication:
1. Sources marks the check as timed out
2. Source status set to `unavailable`
3. Error message: "Availability check timed out after 15 minutes"
4. Pending check record deleted from tracking table

**Why 15 minutes?**
- Most checks complete in <10 seconds
- Allows for application consumer lag or restarts
- Prevents indefinite waiting for lost messages

### Duplicate Handling

Applications should implement idempotent processing:
- If same `correlation_id` received twice → process once, ignore duplicate
- Use database unique constraint on `correlation_id` (if persisting requests)

---

## Implementation Guidelines

### For Producers (Sources)

**Publishing Request:**

```go
// Go example (sources-api-go)
func PublishAvailabilityCheckRequest(sourceID string, appID string, appType string) error {
    correlationID := uuid.New().String()
    
    message := AvailabilityCheckRequest{
        SourceID:        sourceID,
        TenantID:        getTenantID(),
        ApplicationID:   appID,
        ApplicationType: appType,
        SourceType:      getSourceType(sourceID),
        Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
        CorrelationID:   correlationID,
    }
    
    messageBytes, _ := json.Marshal(message)
    
    kafkaMessage := kafka.Message{
        Key:   []byte(sourceID),  // Partition by source_id
        Value: messageBytes,
        Headers: []kafka.Header{
            {Key: "event_type", Value: []byte("availability-check-request")},
            {Key: "schema_version", Value: []byte("1.0")},
            {Key: "tenant_id", Value: []byte(message.TenantID)},
        },
    }
    
    return kafkaWriter.WriteMessages(context.Background(), kafkaMessage)
}
```

**Consuming Result:**

```go
// Go example (sources-api-go)
func ConsumeAvailabilityCheckResult(msg kafka.Message) error {
    var result AvailabilityCheckResult
    json.Unmarshal(msg.Value, &result)
    
    // Update database
    updateSourceAvailabilityStatus(result.SourceID, result.Status, result.Error)
    
    // Publish notification if status changed
    if statusChanged(result.SourceID, result.Status) {
        publishNotificationEvent(result.SourceID, result.Status)
    }
    
    // Delete pending check record
    deletePendingCheck(result.CorrelationID)
    
    return nil
}
```

### For Consumers (Applications)

**Consuming Request:**

```python
# Python example (Cost Management)
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'platform.sources.availability-check-requests',
    group_id='cost-management-availability-consumer',
    auto_offset_reset='latest',
    enable_auto_commit=False
)

for message in consumer:
    request = json.loads(message.value)
    
    # Filter by application_type
    if request['application_type'] != '/insights/platform/cost-management':
        consumer.commit()
        continue
    
    # Process check
    result = perform_availability_check(
        source_id=request['source_id'],
        source_type=request['source_type']
    )
    
    # Publish result
    publish_result(
        source_id=request['source_id'],
        application_id=request['application_id'],
        correlation_id=request['correlation_id'],
        status=result['status'],
        error=result.get('error')
    )
    
    consumer.commit()
```

**Publishing Result:**

```python
# Python example (Cost Management)
from kafka import KafkaProducer
import json
from datetime import datetime

producer = KafkaProducer(
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

def publish_result(source_id, application_id, correlation_id, status, error=None, check_duration_ms=None):
    result = {
        'source_id': source_id,
        'application_id': application_id,
        'status': status,
        'error': error,
        'error_code': categorize_error(error) if error else None,
        'checked_at': datetime.utcnow().isoformat() + 'Z',
        'correlation_id': correlation_id,
        'check_duration_ms': check_duration_ms,
    }
    
    producer.send(
        'platform.sources.availability-check-results',
        key=source_id.encode('utf-8'),
        value=result,
        headers=[
            ('event_type', b'availability-check-result'),
            ('schema_version', b'1.0'),
            ('correlation_id', correlation_id.encode('utf-8')),
        ]
    )
```

---

## Monitoring and Observability

### Metrics to Track

**Producer (Sources):**
- `availability_check_requests_published_total` — Counter of requests published
- `availability_check_publish_errors_total` — Counter of publish failures
- `availability_check_publish_duration_seconds` — Histogram of publish latency

**Consumer (Applications):**
- `availability_check_requests_received_total` — Counter of requests received
- `availability_check_results_published_total` — Counter of results published
- `availability_check_processing_duration_seconds` — Histogram of check execution time
- `availability_check_consumer_lag` — Gauge of consumer lag (messages behind)

**End-to-End:**
- `availability_check_correlation_duration_seconds` — Histogram of request→result latency

### Distributed Tracing

Include `correlation_id` in all logs for tracing:

```
[INFO] Published availability check request: source_id=12345, correlation_id=a1b2c3d4-...
[INFO] Received availability check request: source_id=12345, correlation_id=a1b2c3d4-...
[INFO] Completed availability check: source_id=12345, correlation_id=a1b2c3d4-..., status=available, duration=3245ms
[INFO] Received availability check result: source_id=12345, correlation_id=a1b2c3d4-..., status=available
```

---

## Schema Evolution

### Versioning Strategy

- Schema version included in Kafka headers: `schema_version: "1.0"`
- Backward compatibility required for all changes
- Breaking changes require new topic or major version bump

### Allowed Changes (Backward Compatible)

✅ **Safe:**
- Add new optional field
- Add new enum value to `status` (applications should handle unknown gracefully)
- Add new error code
- Expand string length limits

❌ **Breaking:**
- Remove required field
- Rename field
- Change field type
- Make optional field required

### Example Evolution: Adding `priority` Field

**Version 1.0 (current):**
```json
{
  "source_id": "12345",
  "application_type": "/insights/platform/cost-management",
  // ... other fields
}
```

**Version 1.1 (future - backward compatible):**
```json
{
  "source_id": "12345",
  "application_type": "/insights/platform/cost-management",
  "priority": "high",  // NEW optional field
  // ... other fields
}
```

**Consumer handling:**
```python
priority = request.get('priority', 'normal')  # Default if not present
```

---

## Testing

### Unit Tests

Test message serialization/deserialization:

```go
func TestAvailabilityCheckRequestSerialization(t *testing.T) {
    request := AvailabilityCheckRequest{
        SourceID:        "12345",
        TenantID:        "550e8400-e29b-41d4-a716-446655440000",
        ApplicationID:   "67890",
        ApplicationType: "/insights/platform/cost-management",
        SourceType:      "amazon",
        Timestamp:       "2026-10-07T14:30:00.000Z",
        CorrelationID:   "a1b2c3d4-e5f6-7890-1234-567890abcdef",
    }
    
    bytes, err := json.Marshal(request)
    assert.NoError(t, err)
    
    var decoded AvailabilityCheckRequest
    err = json.Unmarshal(bytes, &decoded)
    assert.NoError(t, err)
    assert.Equal(t, request.SourceID, decoded.SourceID)
}
```

### Integration Tests

Test end-to-end flow using embedded Kafka (testcontainers):

```go
func TestAvailabilityCheckFlow(t *testing.T) {
    // 1. Publish request
    correlationID := publishRequest(sourceID, appID, appType)
    
    // 2. Verify request arrives in topic
    request := consumeRequest()
    assert.Equal(t, correlationID, request.CorrelationID)
    
    // 3. Simulate application publishing result
    publishResult(sourceID, appID, correlationID, "available")
    
    // 4. Verify result processed
    source := getSource(sourceID)
    assert.Equal(t, "available", source.AvailabilityStatus)
}
```

### Schema Validation

Use JSON Schema validators to ensure messages conform:

```python
import jsonschema

schema = load_json_schema('availability-check-request.schema.json')
message = json.loads(kafka_message.value)

try:
    jsonschema.validate(instance=message, schema=schema)
except jsonschema.ValidationError as e:
    logger.error(f"Invalid message format: {e.message}")
    send_to_dead_letter_queue(kafka_message)
```

---

## Error Handling

### Producer Errors (Sources)

**Scenario:** Kafka publish fails

**Handling:**
1. Log error with context (source_id, correlation_id, error message)
2. Mark check as failed in database (don't block user)
3. Return HTTP 202 Accepted (user doesn't see internal error)
4. Metric: `availability_check_publish_errors_total++`

```go
err := kafkaWriter.WriteMessages(ctx, message)
if err != nil {
    log.Error("Failed to publish availability check request", 
        "source_id", sourceID, 
        "error", err)
    
    // Mark as failed in DB
    updateSourceStatus(sourceID, "unavailable", 
        "Failed to trigger availability check")
    
    publishErrorsTotal.Inc()
}
```

### Consumer Errors (Applications)

**Scenario:** Invalid message format

**Handling:**
1. Log error with correlation_id
2. Send to dead letter queue (DLQ) for manual review
3. Commit offset (don't block consumer)
4. Metric: `invalid_message_format_total++`

**Scenario:** Check execution fails (exception)

**Handling:**
1. Catch exception
2. Publish result with `status=unavailable`, `error=exception message`
3. Commit offset
4. Metric: `availability_check_execution_errors_total++`

```python
try:
    status = perform_cloud_provider_check(source_id)
except Exception as e:
    logger.error(f"Check failed: {e}", extra={'correlation_id': correlation_id})
    
    publish_result(
        source_id=source_id,
        status='unavailable',
        error=str(e),
        error_code='UNKNOWN'
    )
    
    execution_errors_total.inc()
```

---

## Security Considerations

### Data Sensitivity

**Non-Sensitive Data in Messages:**
- source_id (internal ID, not PII)
- application_id (internal ID)
- application_type (public path)
- source_type (cloud provider name)
- timestamp, correlation_id (metadata)

**Potentially Sensitive Data:**
- `tenant_id` — Organization ID (not PII, but tenant-specific)
- `source_name` — May contain account numbers or customer naming (optional field)
- `error` — May contain AWS account IDs, resource ARNs (sanitize if needed)

**Not Included in Messages (by design):**
- Credentials (passwords, API keys, IAM roles)
- Full ARNs or resource paths
- Customer PII

### Access Control

**Kafka ACLs:**
- Sources producer: Write access to `availability-check-requests`
- Sources consumer: Read access to `availability-check-results`
- Application consumers: Read access to `availability-check-requests`
- Application producers: Write access to `availability-check-results`

**Multi-Tenancy:**
- Applications must filter by `tenant_id` to ensure they only process their own organization's checks
- Kafka headers include `tenant_id` for filtering

---

## Consumer Implementation

Applications should implement Kafka consumers that:
1. Subscribe to `platform.sources.availability-check-requests` topic
2. Filter messages by their `application_type`
3. Perform availability checks using the source information provided
4. Publish results to `platform.sources.availability-check-results` topic

See the [Implementation Guidelines](#implementation-guidelines) section for code examples.

### Application Team Requirements

Each application consuming availability check requests must:

#### Cost Management
- **Consumer group:** `cost-management-availability-consumer`
- **Filter by:** `application_type = "/insights/platform/cost-management"`
- **Check types:** AWS, Azure, GCP, OpenShift, IBM cloud provider credentials
- **Implementation:** Add consumer to `clowder-sources-listener` or create dedicated consumer

#### cloud-meter / rhsm-api-proxy
- **Consumer group:** `cloud-meter-availability-consumer`
- **Filter by:** `application_type = "/insights/platform/cloud-meter"`
- **Check types:** AWS, Azure, GCP cloud provider credentials for RHEL management
- **Implementation:** Add consumer (likely in rhsm-api-proxy component)

#### Cloud Connector
- **Consumer group:** `cloud-connector-availability-consumer`
- **Filter by:** `application_type = "cloud-connector"` or `source_type = "satellite"`
- **Check types:** Red Hat Connector (RHC) connection status checks
- **Implementation:** Go-based consumer using existing connection status logic

#### Provisioning
- **Consumer group:** `provisioning-availability-consumer`
- **Filter by:** `application_type = "/insights/platform/provisioning"`
- **Check types:** Infrastructure provisioning credentials
- **Implementation:** Add consumer (if service is active)

---

## Questions and Feedback

### Review Process

This schema design must be reviewed by:
- ✅ Sources team (architecture review)
- ⬜ Cost Management team (schema validation, feasibility)
- ⬜ cloud-meter team (schema validation, feasibility)
- ⬜ Cloud Connector team (schema validation, RHC-specific needs)
- ⬜ Provisioning team (active/disabled status, schema validation if active)

### Feedback Channels

**Questions or concerns?**
1. Comment on JIRA ticket: RHCLOUD-52010
2. Open an issue in the [sources-api-go repository](https://github.com/RedHatInsights/sources-api-go/issues)
3. Contact the Sources team via your organization's communication channels

**Schema change requests:**
1. Open PR against this document
2. Tag reviewers from all application teams
3. Update JSON Schema files

---

## References

- **JSON Schema Spec:** [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core.html)
- **Migration Context:** [Architecture Evaluation](../../../availability-check-architecture-evaluation.md) (RHCLOUD-44007)

---

**Document Version:** 1.0  
**Last Updated:** 2026-10-07  
**Status:** Draft — Pending Application Team Review
