# JWT Authentication Setup

This document explains how to configure and use JWT-based OIDC authentication in the Sources API.

## Overview

JWT authentication has been added as an additional authentication method that works alongside existing PSK and x-rh-identity authentication. This enables secure Service-to-Service (S2S) communication using industry-standard OIDC JWT tokens with JWKS-based key discovery.

## Configuration

### Environment Variables

JWT authentication uses OIDC Discovery for automatic JWKS endpoint discovery:

```bash
# Required: JWT issuer URL for OIDC discovery
JWT_ISSUER="https://your-oidc-provider"
```

### Feature Flag

JWT authentication is controlled by a feature flag:

```bash
# Feature flag name
"sources-api.oidc-auth.enabled"
```

## Authentication Flow

The API processes authentication in the following order:

### Phase 1: JWT Authentication Middleware
- Extracts JWT from `Authorization: Bearer <token>` header (processed by ParseHeaders middleware)
- If feature flag `sources-api.oidc-auth.enabled` is disabled, skips JWT processing
- If no token present, continues to next authentication method
- Discovers JWKS URL from issuer's OIDC discovery endpoint
- Validates token signature using discovered JWKS
- Extracts and stores JWT subject in context

### Phase 2: Authorization Middleware
Checks authorization in this order:
1. **PSK Authentication** - `x-rh-sources-psk` header
2. **JWT Authentication** - Subject from validated JWT token
3. **x-rh-identity** - Red Hat Identity header
4. **Unauthorized** - If none of the above are present/valid

## JWT Token Requirements

Your JWT token must:

- Be provided in `Authorization: Bearer <token>` header
- Be signed with a cryptographic algorithm supported by your issuer
- Include a `sub` (subject) claim with the user/service identifier (max 256 bytes)
- Include `exp` (expiration) claim
- Include `iat` (issued at) claim
- Be verifiable using public keys from the configured JWKS endpoint

### Security Features

- **Clock Skew Tolerance**: 30 seconds tolerance for time-based claims
- **Subject Length Limits**: Maximum 256 bytes to prevent memory exhaustion attacks
- **Timeout Protection**: 10-second timeout for token validation
- **OIDC Discovery**: Automatic JWKS URL discovery (fetched every time JWKS is refreshed)
- **JWKS Caching**: 10-minute cache with async refresh and fallback protection


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

## Usage

Send requests with the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/api/sources/v3.1/sources
```

## OIDC Discovery Requirements

Your OIDC provider must:

### Discovery Endpoint
- Provide discovery document at `https://your-issuer/.well-known/openid-configuration`
- Return HTTP 200 status code with `Content-Type: application/json`
- Include required fields: `issuer` and `jwks_uri`
- Be accessible over HTTPS (HTTP only allowed for localhost in test environments)
- Respond within 8 seconds

### JWKS Endpoint
- Return HTTP 200 status code
- Use `Content-Type: application/json`
- Return valid JWKS JSON format
- Contain at least one valid cryptographic key
- Be accessible over HTTPS (HTTP only allowed for localhost in test environments)
- Respond within 8 seconds

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

Run the JWT authentication tests:

```bash
# Test JWT authentication middleware
go test ./middleware -v -run TestJWTAuthentication

# Test JWT validation functions
go test ./middleware -v -run TestValidateJWT

# Test JWKS functionality (includes async refresh tests)
go test ./middleware -v -run TestFetchJWKS

# Test JWKS security validation
go test ./middleware -v -run TestSecureJWKSFetch

# Test OIDC discovery functionality
go test ./middleware -v -run TestDiscoverJWKSURL

# Test discovery URL building and validation
go test ./middleware -v -run TestBuildDiscoveryURL

# Test discovery document fetching
go test ./middleware -v -run TestPerformDiscovery
```

## Error Handling

Common error responses:

- `401 Unauthorized` with "Authentication failed" - Token validation failed
- `401 Unauthorized` - JWKS fetch failed or token parsing failed
- HTTP 200 with no JWT subject set - Feature flag disabled or no token provided

## Implementation Details

### Key Files

- `middleware/jwt_auth.go` - Main JWT authentication middleware and validation
- `middleware/jwks_discovery.go` - OIDC discovery document fetching
- `middleware/jwks.go` - JWKS fetching, caching, and key validation with dependency injection
- `middleware/authorization.go` - Authorization logic integrating JWT subjects
- `config/config.go` - Configuration management for JWT_ISSUER

### Architecture

- **JWTAuthentication()** - Echo middleware function
- **validateJWTToken()** - Core token validation with JWKS
- **validateJWTSubject()** - Subject claim validation
- **discoverJWKSURL()** - OIDC discovery function (in jwks_discovery.go)
- **FetchJWKS()** - JWKS fetching with caching (in jwks.go)
- **FetchJWKSWithDiscoverer()** - JWKS fetching with dependency injection support
- **refreshJWKSAsync()** - Background JWKS refresh functionality

### Caching Behavior

The implementation uses two separate caches for optimal performance:

#### OIDC Discovery (No Caching)
1. **Fresh Discovery**: Fetches discovery document every time JWKS needs to be refreshed
2. **Simplified Logic**: No cache management complexity
3. **Always Current**: Uses latest discovery information from IdP
4. **Reasonable Load**: ~144 discovery calls per day (every 10 minutes)

#### JWKS Cache (10-minute TTL)
1. **Cache Hit (Fresh)**: Returns cached JWKS immediately if within 10-minute TTL
2. **Cache Hit (Expired)**: Returns cached JWKS immediately and triggers background refresh
3. **Cache Miss**: Fetches JWKS synchronously using discovered URL and caches result
4. **Fetch Failure**: Falls back to cached JWKS to prevent service outages
5. **Async Refresh**: Background refresh updates JWKS cache without blocking requests

This design prioritizes availability over freshness for JWKS data, ensuring that IdP outages don't impact authentication.

### Dependency Injection Architecture

The implementation uses interface-based dependency injection for testability:

#### JWKSDiscoverer Interface
```go
type JWKSDiscoverer interface {
    Discover(ctx context.Context) (string, error)
}
```

#### Production Implementation
- **DefaultJWKSDiscoverer**: Production implementation using real OIDC discovery
- **FetchJWKS()**: Uses default discoverer for normal operation
- **FetchJWKSWithDiscoverer()**: Accepts custom discoverer for testing

#### Testing Implementation
- **mockJWKSDiscoverer**: Test implementation for controlled testing
- **newMockJWKSDiscoverer()**: Helper to create mock discoverers with specified URLs/errors
- Enables isolated testing without real HTTP calls or OIDC discovery
- Allows testing error conditions, timeouts, and various JWKS URL scenarios
- Separates JWKS fetching tests from OIDC discovery tests for better test isolation

### Performance Optimizations

- **JWKS Caching**: 10-minute cache to reduce JWKS endpoint calls
- **Async Refresh**: Background refresh for JWKS to avoid blocking requests
- **Fallback Protection**: Returns cached JWKS on fetch failures to prevent outages
- **Double-Checked Locking**: Optimized cache access with minimal lock contention
- **Timeout Protection**: Request-scoped timeouts prevent hanging
- **Simplified Discovery**: No discovery caching reduces complexity while maintaining reasonable load
- **Dependency Injection**: Interface-based design enables clean testing with mock discoverers

## Security Considerations

- **Discovery Security**: OIDC discovery endpoint must be trusted and secured
- **JWKS Security**: JWKS endpoint must be trusted and secured
- **Subject Validation**: Length limits prevent memory exhaustion
- **Network Security**: HTTPS required for discovery and JWKS endpoints (except localhost in tests)
- **Timeout Protection**: Prevents hanging on slow/malicious endpoints
- **Issuer Validation**: Discovery document issuer must match requested issuer

## Migration from Previous Setup

If migrating from a previous JWT implementation:

1. Remove any static JWT public key configuration
2. Remove `JWKS_URL` environment variable (no longer used)
3. Configure `JWT_ISSUER` environment variable with your OIDC provider's issuer URL
4. Ensure your OIDC provider supports discovery at `/.well-known/openid-configuration`
5. Enable feature flag `sources-api.oidc-auth.enabled`
6. Update JWT tokens to include required claims (`sub`, `exp`, `iat`)

For questions or issues, please refer to the test files for examples of proper usage.
