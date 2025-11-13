# JWT Authentication Setup

This document explains how to configure and use JWT-based OIDC authentication in the Sources API.

## Overview

JWT authentication has been added as an additional authentication method that works alongside existing PSK and x-rh-identity authentication. This enables secure Service-to-Service (S2S) communication using industry-standard OIDC JWT tokens with automatic public key retrieval from your OIDC provider's JWKS endpoint.

**Note**: JWT authentication is available on **both public and internal API endpoints** for complete feature parity with PSK authentication.

## Architecture Overview

The JWT authentication system consists of three main components:

1. **JWT Authentication Middleware** (`middleware/jwt_auth.go`)
   - Validates JWT tokens using two-step process (issuer check → signature verification)
   - Checks if issuer/subject pair is whitelisted in `AUTHORIZED_JWT_SUBJECTS`
   - Sets context values for successful JWT authentication or returns 401/403 errors

2. **JWKS Management** (`middleware/jwks.go`)
   - Automatic OIDC discovery for JWKS URL discovery
   - Two-level caching system for optimal performance
   - Stale-while-revalidate pattern for high availability

3. **Route Integration** (`routes.go`)
   - JWT middleware applied to both public and internal API routes

## Configuration

### Environment Variables

JWT authentication uses OIDC Discovery for automatic JWKS endpoint discovery:

```bash
# Required: JWT issuer URL for OIDC discovery
JWT_ISSUER="https://your-oidc-provider"

# Optional: JSON array of authorized issuer-subject pairs for additional authorization
AUTHORIZED_JWT_SUBJECTS='[{"issuer":"https://your-oidc-provider", "subject":"service-user-123"}]'

# Optional: For testing with localhost OIDC providers
GO_ENV="test"  # Allows HTTP for localhost/127.0.0.1
```

### Feature Flag

JWT authentication is controlled by a feature flag:

```bash
# Feature flag name
"sources-api.oidc-auth.enabled"
```

When disabled, JWT middleware is completely bypassed, allowing fallback to PSK authentication.

## Authentication Flow

### Phase 1: JWT Authentication Middleware (All APIs)

Applied to both `/api/sources/*` and `/internal/*` routes in this order:

1. **Feature Flag Check**: If `sources-api.oidc-auth.enabled` is disabled, skips JWT processing
2. **Token Extraction**: Extracts JWT from `Authorization: Bearer <token>` header
3. **No Token Handling**: If no token present, continues to next authentication method (PSK)
4. **JWT Validation**:
   - **Step 1**: Parse token and validate issuer against `JWT_ISSUER` config
   - **Step 2**: Fetch JWKS and verify signature + claims (exp, sub, iat)
5. **Authorization Check**: Validates issuer/subject pair against `AUTHORIZED_JWT_SUBJECTS` whitelist
6. **Result**: Either sets context values (success) or returns 401/403 (failure)

### Phase 2: Permission Check Middleware

For **both public and internal APIs**, checks authorization in this order:
1. **JWT Authentication** - Uses JWT context values set by JWT middleware (if present)
2. **PSK Authentication** - `x-rh-sources-psk` header  
3. **RBAC Authentication** - `x-rh-identity` header (public APIs only)
4. **Unauthorized** - If no valid authentication method is present

**Note**: JWT whitelist checking happens in Phase 1 (JWT middleware), not in Permission Check middleware.

### Phase 3: User Management Middleware

The `UserCatcher` middleware currently handles:
- **PSK Authentication**: Uses `x-rh-sources-user-id` header to create/find users in database
- **RBAC Authentication**: Uses `x-rh-identity` header to extract user ID and create/find users in database
- **JWT Authentication**: Does **not** create users in database (JWT context values are not processed by UserCatcher)

#### JWT Subject Authorization (Phase 1)

JWT subject authorization happens in the JWT middleware itself:
- JWT must have both valid issuer and subject claims
- The issuer-subject pair must match one of the configured authorized pairs in `AUTHORIZED_JWT_SUBJECTS`
- If `AUTHORIZED_JWT_SUBJECTS` is not set or empty, **all JWT tokens will be rejected** (secure by default)
- Authorization check occurs immediately after JWT validation, before reaching Permission Check middleware

## Caching Architecture

The implementation uses a sophisticated two-level caching system:

### 1. Discovery Cache (1-hour TTL)
- **Purpose**: Caches JWKS URLs discovered via OIDC discovery
- **TTL**: 1 hour with stale-while-revalidate pattern
- **Cache Miss**: Discovers JWKS URL synchronously, caches result
- **Cache Hit (Fresh)**: Uses cached JWKS URL immediately
- **Cache Hit (Expired)**: Uses stale JWKS URL immediately, triggers async refresh
- **Size**: LRU cache with 10 issuer capacity

### 2. JWKS Content Cache (1-hour refresh)
- **Purpose**: Caches actual JWKS documents with automatic background refresh
- **Refresh**: 1-hour refresh interval with automatic background updates
- **Registration**: JWKS URLs are registered once, then auto-refreshed
- **Fallback**: Uses cached JWKS on fetch failures to prevent service outages

This design prioritizes **availability over freshness**, ensuring that IdP outages don't impact authentication.

## API Coverage

### Public APIs (JWT + PSK + RBAC Support)
```
/api/sources/v1.0/*
/api/sources/v2.0/*
/api/sources/v3.0/*
/api/sources/v3.1/*
/api/sources/v1/*
/api/sources/v2/*
/api/sources/v3/*
```

**Authentication methods supported**:
- JWT tokens (with `Authorization: Bearer` header)
- PSK authentication (with `x-rh-sources-psk` header)
- RBAC authentication (with `x-rh-identity` header)

### Internal APIs (JWT + PSK Support)
```
/internal/v1.0/*
/internal/v2.0/*
```

**Authentication methods supported**:
- JWT tokens (with `Authorization: Bearer` header)
- PSK authentication (with `x-rh-sources-psk` header)

**Supported endpoints**:
- `GET /internal/v1.0/authentications/:uuid`
- `GET /internal/v1.0/secrets/:id`
- `GET /internal/v1.0/sources`
- `GET /internal/v1.0/untranslated-tenants`
- `POST /internal/v1.0/translate-tenants`

## JWT Token Requirements

Your JWT token must:

- Be provided in `Authorization: Bearer <token>` header
- Be signed with a cryptographic algorithm supported by your issuer
- Include required claims:
  - `iss` (issuer) - Must match `JWT_ISSUER` configuration
  - `sub` (subject) - User/service identifier (max length validated)
  - `exp` (expiration) - Token expiration time
  - `iat` (issued at) - Token issuance time
- Be verifiable using public keys from the JWKS endpoint

### Security Features

- **Two-Step Validation**: Issuer check before expensive JWKS operation
- **Clock Skew Tolerance**: 30 seconds tolerance for time-based claims
- **Timeout Protection**: 10-second timeout for token validation
- **Signature Verification**: Full cryptographic verification using JWKS
- **Claims Validation**: Validates exp, sub, iat claims

### Example JWT Claims

```json
{
  "sub": "service-user-123",
  "exp": 1735689600,
  "iat": 1735686000,
  "iss": "https://your-oidc-provider",
  "aud": "your-audience"
}
```

## Usage Examples

### Public API Authentication

```bash
# Option 1: JWT authentication (preferred for service-to-service)
curl -H "Authorization: Bearer <your-jwt-token>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/api/sources/v3.1/sources

# Option 2: PSK authentication
curl -H "x-rh-sources-psk: <your-psk>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/api/sources/v3.1/sources

# Option 3: RBAC authentication
curl -H "x-rh-identity: <base64-encoded-identity>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/api/sources/v3.1/sources
```

### Internal API Authentication

```bash
# Option 1: JWT authentication (preferred for service-to-service)
curl -H "Authorization: Bearer <your-jwt-token>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/internal/v1.0/sources

# Option 2: PSK authentication
curl -H "x-rh-sources-psk: <your-psk>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/internal/v1.0/sources
```

## OIDC Discovery Requirements

Your OIDC provider must support:

### Discovery Endpoint
- **URL**: `https://your-issuer/.well-known/openid-configuration`
- **HTTP Status**: 200 OK
- **Content-Type**: `application/json` (with optional charset)
- **Required Fields**: `issuer` and `jwks_uri`
- **Security**: HTTPS required (HTTP allowed for localhost in test environments)
- **Timeout**: Must respond within 8 seconds
- **Issuer Validation**: Discovery document `issuer` field must match requested issuer

### JWKS Endpoint
- **HTTP Status**: 200 OK
- **Content-Type**: `application/json`
- **Format**: Valid JWKS JSON format
- **Content**: At least one valid cryptographic key
- **Security**: HTTPS required (HTTP allowed for localhost in test environments)
- **Timeout**: Must respond within 8 seconds

### Example Discovery Document

```json
{
  "issuer": "https://your-oidc-provider",
  "jwks_uri": "https://your-oidc-provider/.well-known/jwks.json",
  "authorization_endpoint": "https://your-oidc-provider/oauth/authorize",
  "token_endpoint": "https://your-oidc-provider/oauth/token"
}
```

### Example JWKS Response

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-1",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

## Testing

### Unit Tests

```bash
# JWT authentication middleware tests (data-driven)
go test ./middleware -v -run TestJWTAuthentication_AllScenarios

# JWT validation function tests (data-driven) 
go test ./middleware -v -run TestValidateJWT_AllScenarios

# JWT token extraction tests
go test ./middleware -v -run TestExtractJWT

# JWT subject authorization tests
go test ./middleware -v -run TestIsJWTSubjectAuthorized
```

### JWKS Unit Tests

```bash
# JWKS function mocking capability
go test ./middleware -v -run TestGetJWKS_Integration

# Discovery URL building and validation
go test ./middleware -v -run TestBuildDiscoveryURL

# OIDC discovery document processing
go test ./middleware -v -run TestPerformDiscovery

# Complete JWKS discovery workflow
go test ./middleware -v -run TestDiscoverJWKSURL
```

### Integration Tests

```bash
# End-to-end JWKS scenarios (data-driven)
go test ./middleware -v -run TestGetJWKS_AllScenarios

# JWKS caching behavior 
go test ./middleware -v -run TestGetJWKS_CachingBehavior

# Discovery URL scenarios (data-driven)
go test ./middleware -v -run TestBuildDiscoveryURL_AllScenarios

# Complete discovery workflow (data-driven)
go test ./middleware -v -run TestDiscoverJWKSURL_AllScenarios

# OIDC discovery error scenarios (data-driven)
go test ./middleware -v -run TestPerformDiscovery_AllScenarios

# Cache expiration logic
go test ./middleware -v -run TestCachedJWKSURL_IsExpired
```

### Run All JWT Tests

```bash
go test ./middleware -v -run "(JWT|JWKS|Discovery|Cache)"
```

## Error Handling

### HTTP Response Codes

- **401 Unauthorized** with "Authentication failed"
  - JWT parsing failed
  - Invalid issuer
  - Signature verification failed
  - JWKS retrieval failed
  - Missing or invalid claims

- **403 Forbidden** with "JWT subject not authorized"  
  - Valid JWT but subject not in `AUTHORIZED_JWT_SUBJECTS`

- **200 OK** (JWT middleware completes successfully)
  - Feature flag disabled - JWT middleware skipped entirely
  - No JWT token provided - JWT middleware passes through to next middleware
  - Valid JWT with authorized subject - JWT middleware sets context and continues

### Common Issues

1. **"JWKS URL discovery failed"** - OIDC discovery endpoint issues
2. **"JWKS fetch failed"** - JWKS endpoint unavailable or invalid
3. **"JWT validation failed"** - Token signature or claims invalid
4. **"issuer mismatch"** - JWT issuer doesn't match `JWT_ISSUER` config

## Implementation Details

### Key Files

- `middleware/jwt_auth.go` - JWT authentication middleware and validation logic
- `middleware/jwks.go` - JWKS fetching, OIDC discovery, and two-level caching system
- `middleware/jwt_auth_test.go` - Comprehensive data-driven JWT authentication tests
- `middleware/jwks_test.go` - JWKS unit tests with mocking support
- `middleware/jwks_integration_test.go` - End-to-end JWKS integration tests
- `routes.go` - JWT middleware integration (both public and internal APIs)
- `config/config.go` - Configuration management for JWT settings

### Performance Optimizations

- **Two-Level Caching**: Discovery cache (1-hour) + JWKS cache (1-hour refresh)
- **Stale-While-Revalidate**: Uses expired cache immediately, refreshes in background
- **Async Refresh**: Background JWKS refresh doesn't block requests
- **Early Issuer Check**: Validates issuer before expensive JWKS operations
- **Connection Pooling**: Reuses HTTP connections for OIDC discovery and JWKS
- **Timeout Protection**: Request-scoped timeouts prevent hanging
- **LRU Eviction**: Discovery cache uses LRU eviction for memory management

## Security Considerations

### Network Security
- **HTTPS Required**: Discovery and JWKS endpoints must use HTTPS (except localhost in tests)
- **Timeout Protection**: 8-second timeouts prevent hanging on slow/malicious endpoints
- **Certificate Validation**: Full TLS certificate validation for HTTPS endpoints

### Token Security  
- **Two-Step Validation**: Issuer validation before expensive cryptographic operations
- **Signature Verification**: Full cryptographic verification using JWKS public keys
- **Claims Validation**: Validates exp, sub, iat claims with clock skew tolerance
- **Issuer Validation**: Discovery document issuer must match requested issuer

### Authorization Security
- **Secure by Default**: Empty `AUTHORIZED_JWT_SUBJECTS` rejects all JWTs
- **Explicit Allow-listing**: Only configured issuer-subject pairs are authorized
- **Multiple Authentication Options**: JWT available alongside PSK and RBAC methods
- **JWT User Management**: JWT authentication does not create users in the database (handled separately from PSK/RBAC user management)

### Availability Security
- **Graceful Degradation**: All APIs fall back to other auth methods if JWT fails
- **Multiple Auth Options**: Public APIs support JWT, PSK, and RBAC; Internal APIs support JWT and PSK
- **Cache Fallback**: Uses stale JWKS cache during IdP outages
- **Feature Flag Control**: Can disable JWT auth instantly if needed
- **Non-blocking Design**: JWT middleware doesn't block other auth methods

For questions or issues, please refer to the comprehensive test files for examples of proper usage and integration patterns.
