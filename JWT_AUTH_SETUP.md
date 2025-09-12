# JWT Authentication Setup

This document explains how to configure and use JWT-based OIDC authentication in the Sources API.

## Overview

JWT authentication has been added as an additional authentication method that works alongside existing PSK and x-rh-identity authentication. This enables secure Service-to-Service (S2S) communication using industry-standard OIDC JWT tokens with JWKS-based key discovery.

## Configuration

### Environment Variables

JWT authentication requires JWKS (JSON Web Key Set) for dynamic key discovery:

```bash
# Required: JWKS endpoint URL for key discovery
JWKS_URL="https://your-oidc-provider/.well-known/jwks.json"
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
- Validates token signature using JWKS
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
- Be signed with an **allowed algorithm**: RS256, RS384, RS512, ES256, ES384, ES512
- Include a `sub` (subject) claim with the user/service identifier (max 256 bytes)
- Include `exp` (expiration) claim
- Include `iat` (issued at) claim
- Be verifiable using public keys from the configured JWKS endpoint

### Security Features

- **Algorithm Allowlist**: Only secure asymmetric algorithms are allowed
- **Clock Skew Tolerance**: 30 seconds tolerance for time-based claims
- **Subject Length Limits**: Maximum 256 bytes to prevent memory exhaustion attacks
- **Timeout Protection**: 10-second timeout for token validation
- **JWKS Caching**: 10-minute cache with async refresh and fallback protection

### Rejected Algorithms (Security)

The following algorithms are explicitly rejected for security reasons:
- **Symmetric algorithms**: HS256, HS384, HS512 (shared secret vulnerabilities)
- **None algorithm**: `none` (no signature)
- **PSS algorithms**: PS256, PS384, PS512 (complexity)
- **EdDSA**: Not commonly used in OIDC
- **ES256K**: Bitcoin-specific variant

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

## JWKS Requirements

Your JWKS endpoint must:

- Return HTTP 200 status code
- Use `Content-Type: application/json`
- Return valid JWKS JSON format
- Contain at least one valid RSA key
- Have RSA keys with minimum 2048-bit strength
- Be accessible over HTTPS (HTTP only allowed for localhost in test environments)
- Respond within 8 seconds
- Return responses under 32KB

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
```

## Error Handling

Common error responses:

- `401 Unauthorized` with "Authentication failed" - Token validation failed
- `401 Unauthorized` - JWKS fetch failed or token parsing failed
- HTTP 200 with no JWT subject set - Feature flag disabled or no token provided

## Implementation Details

### Key Files

- `middleware/jwt_auth.go` - Main JWT authentication middleware and validation
- `middleware/jwks.go` - JWKS fetching, caching, and key validation
- `middleware/authorization.go` - Authorization logic integrating JWT subjects
- `config/config.go` - Configuration management for JWKS_URL

### Architecture

- **JWTAuthentication()** - Echo middleware function
- **validateJWTToken()** - Core token validation with JWKS
- **validateJWTAlgorithm()** - Algorithm security validation
- **validateJWTSubject()** - Subject claim validation
- **FetchJWKS()** - JWKS discovery with caching
- **refreshJWKSAsync()** - Background JWKS refresh functionality

### JWKS Caching Behavior

The JWKS cache implements sophisticated caching with async refresh to ensure high availability:

1. **Cache Hit (Fresh)**: Returns cached JWKS immediately if within 10-minute TTL
2. **Cache Hit (Expired)**: Returns cached JWKS immediately and triggers background refresh
3. **Cache Miss**: Fetches JWKS synchronously and caches result
4. **Fetch Failure**: Falls back to cached JWKS to prevent service outages
5. **Async Refresh**: Background refresh updates cache without blocking requests

This design prioritizes availability over freshness, ensuring that IdP outages don't impact authentication.

### Performance Optimizations

- **JWKS Caching**: 10-minute cache to reduce external calls
- **Async Refresh**: Background JWKS refresh to avoid blocking requests
- **Fallback Protection**: Returns cached JWKS on fetch failures to prevent outages
- **Double-Checked Locking**: Optimized cache access with minimal lock contention
- **Timeout Protection**: Request-scoped timeouts prevent hanging
- **Size Limits**: Response size limits prevent DoS attacks

## Security Considerations

- **JWKS Security**: JWKS endpoint must be trusted and secured
- **Algorithm Restrictions**: Only secure asymmetric algorithms allowed
- **Key Strength**: Minimum 2048-bit RSA keys enforced
- **Subject Validation**: Length limits prevent memory exhaustion
- **Network Security**: HTTPS required for JWKS endpoints (except localhost in tests)
- **Timeout Protection**: Prevents hanging on slow/malicious endpoints

## Migration from Previous Setup

If migrating from a previous JWT implementation:

1. Remove any static JWT public key configuration
2. Set up JWKS endpoint with your OIDC provider
3. Configure `JWKS_URL` environment variable
4. Enable feature flag `sources-api.oidc-auth.enabled`
5. Update JWT tokens to include required claims (`sub`, `exp`, `iat`)
6. Ensure JWT signing algorithm is in the allowed list

For questions or issues, please refer to the test files for examples of proper usage.
