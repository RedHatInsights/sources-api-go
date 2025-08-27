# JWT Authentication Setup

This document explains how to configure and use JWT-based Service-to-Service (S2S) authentication in the Sources API.

## Overview

JWT authentication has been added as a fallback authentication method alongside existing PSK and x-rh-identity authentication. This enables secure S2S communication using industry-standard JWT tokens.

## Configuration

## Configuration Options

You can configure JWT authentication using either **static keys** or **JWKS discovery**:

### Option 1: Static Public Key (Simple)

```bash
# Required: RSA public key in PEM format for JWT signature verification
JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----"

# Optional: Expected issuer of the JWT token
JWT_ISSUER="your-service-issuer"

# Optional: Expected audience of the JWT token  
JWT_AUDIENCE="sources-api"
```

### Option 2: JWKS Discovery (Dynamic)

```bash
# Required: JWKS endpoint URL for dynamic key discovery
JWT_JWKS_URL="https://your-auth-server/.well-known/jwks.json"

# Optional: Expected issuer of the JWT token
JWT_ISSUER="your-service-issuer" 

# Optional: Expected audience of the JWT token
JWT_AUDIENCE="sources-api"
```

**Note:** If `JWT_JWKS_URL` is set, it takes precedence over `JWT_PUBLIC_KEY`.

## Authentication Flow

The API checks authentication in the following order:

1. **PSK Authentication** - `x-rh-sources-psk` header
2. **x-rh-identity** - Red Hat Identity header
3. **JWT Authentication** - `Authorization: Bearer <token>` header
4. **Unauthorized** - If none of the above are present/valid

## JWT Token Requirements

Your JWT token must:

- Be signed with RS256 algorithm
- Include a `sub` claim with the user/service identifier
- Include `exp` claim for expiration (tokens are validated for expiration)
- If `JWT_ISSUER` is set, include matching `iss` claim
- If `JWT_AUDIENCE` is set, include matching `aud` claim
- For JWKS: Include `kid` (key ID) in token header to identify the signing key (optional but recommended)

### Example JWT Claims

```json
{
  "sub": "service-user-123",
  "iss": "your-service-issuer",
  "aud": "sources-api", 
  "exp": 1735689600,
  "iat": 1735686000
}
```

## Usage

Send requests with the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
     -H "Content-Type: application/json" \
     https://your-sources-api/api/sources/v1.0/sources
```

## JWKS vs Static Key Comparison

| Feature | Static Key | JWKS |
|---------|------------|------|
| **Setup Complexity** | Simple | Moderate |
| **Key Rotation** | Manual | Automatic |
| **Multiple Keys** | Single key only | Multiple keys supported |
| **Performance** | Fast (no HTTP calls) | Cached (5min TTL) |
| **Dependencies** | Minimal | Requires JWKS endpoint |
| **Security** | Good (manual rotation) | Better (automatic rotation) |

### When to Use Each

- **Static Key**: Simple S2S authentication, controlled environments, single service
- **JWKS**: Multi-service environments, automatic key rotation, enterprise setups

## Testing

Run the JWT authentication tests:

```bash
go test ./middleware -v -run TestJWT
go test ./middleware/oidc -v -run TestValidateJWT
go test ./middleware -v -run TestJWKS
```

## Security Considerations

- **Public Key Security**: Keep your JWT signing private key secure and only configure the public key in the API
- **Token Expiration**: Set reasonable expiration times on JWT tokens (recommended: 1 hour or less)
- **Issuer/Audience Validation**: Use the `JWT_ISSUER` and `JWT_AUDIENCE` environment variables for additional security
- **Algorithm**: Only RS256 (RSA with SHA-256) is supported for security reasons

## Error Handling

Common error responses:

- `401 Unauthorized` with message "Invalid JWT token" - Token validation failed
- `401 Unauthorized` with message "JWT_PUBLIC_KEY not configured" - Missing configuration
- `401 Unauthorized` with message "Invalid JWT claims" - Missing or invalid `sub` claim

## Implementation Details

- JWT validation is implemented in `middleware/jwt_validator.go`
- JWKS client and caching in `middleware/jwks_client.go`
- Authorization logic is in `middleware/authorization.go`
- User extraction is handled in `middleware/user.go`
- Configuration is managed in `config/config.go`

### Key Files

- `jwt_validator.go` - Static key validation, unified validation interface
- `jwks_client.go` - JWKS fetching, caching, and dynamic key discovery
- `authorization.go` - Middleware integration and authentication flow

For questions or issues, please refer to the test files for examples of proper usage. 