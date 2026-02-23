# OpenShift Authentication Plugin - Technical Documentation

This document provides comprehensive technical documentation about the OpenShift Auth Plugin implementation, security considerations, and architecture details.

## Overview

The OpenShift Auth Plugin is a Kubernetes ExecCredential authentication plugin (implements `client.authentication.k8s.io/v1beta1`) that automatically authenticates `kubectl` commands with OpenShift clusters. It's available in two implementations:

- **Go version** (recommended): Self-contained binary with better performance and comprehensive error handling
- **Shell script version** (legacy): Bash script with external dependencies

## Authentication Flow

### Three-Method Authentication (Go Version)

The Go plugin tries three authentication methods in order of precedence:

1. **Cached Token Method** (Primary)
   - Checks secure storage (Keychain or gopass) for cached token
   - Validates token expiry using:
     - Timestamp-based expiry (ISO 8601 UTC format: `2006-01-02T15:04:05Z`)
     - API call to `/apis/user.openshift.io/v1/users/~` endpoint
   - Returns cached token if valid
   - Falls through if token expired or invalid

2. **SSO/PKCE Authentication** (Secondary)
   - Uses OAuth 2.0 Authorization Code + PKCE (RFC 7636)
   - Opens default browser for federated login (Azure AD, Okta, etc.)
   - Starts local callback server on random `127.0.0.1` port
   - Exchanges authorization code for access token using PKCE verification
   - Stores token for future use
   - Falls through if browser unavailable or cluster doesn't support SSO

3. **Username/Password Fallback** (Tertiary)
   - Retrieves stored credentials from secure storage
   - Authenticates using HTTP Basic Auth
   - Client ID: `openshift-challenging-client`
   - Stores returned token for future use
   - Fails with error if no credentials found

### Two-Method Authentication (Shell Script Version)

1. **Cached Token** - Same as Go version
2. **Username/Password** - Same fallback as Go version

### OAuth Client IDs

- **SSO/PKCE Flow**: `openshift-cli-client` (standard OpenShift CLI client)
- **Basic Auth Flow**: `openshift-challenging-client` (HTTP Basic Auth client)

## OpenShift OAuth Endpoints

### OAuth Metadata Discovery

The plugin discovers OAuth endpoints by fetching metadata:

```
GET <OPENSHIFT_URL>/.well-known/oauth-authorization-server
```

Response contains:
```json
{
  "authorization_endpoint": "https://oauth-openshift.apps.example.com/oauth/authorize",
  "token_endpoint": "https://oauth-openshift.apps.example.com/oauth/token",
  "code_challenge_methods_supported": ["S256", "plain"]
}
```

### OAuth URLs by Use Case

**From OpenShift Console**:
```
https://oauth-openshift.apps.example.com/oauth/authorize
  ?client_id=console
  &redirect_uri=https://console-openshift-console.apps.example.com/auth/callback
  &response_type=code
  &scope=user:full
  &state=<random-state>
```

**With IDP Selection (e.g., AzureAD)**:
```
https://oauth-openshift.apps.example.com/oauth/authorize
  ?client_id=console
  &idp=AzureAD
  &redirect_uri=https://console-openshift-console.apps.example.com/auth/callback
  &response_type=code
  &scope=user:full
  &state=<random-state>
```

**From Token Display Page**:
```
https://oauth-openshift.apps.example.com/oauth/authorize
  ?client_id=openshift-browser-client
  &redirect_uri=https://oauth-openshift.apps.example.com/oauth/token/display
  &response_type=code
```

**From CLI (PKCE)**:
```
https://oauth-openshift.apps.example.com/oauth/authorize
  ?client_id=openshift-cli-client
  &code_challenge=D0fMJ61pkLRP3uNVmauOOia-OsP25ndloYWE8INKEr0
  &code_challenge_method=S256
  &redirect_uri=http://127.0.0.1:56079/callback
  &response_type=code
  &state=<random-state>
```

## Security Architecture

### HTTPS Validation

All OAuth endpoints **must use HTTPS** to prevent man-in-the-middle attacks:

```
Authorization Endpoint: https://oauth-openshift.apps.example.com/oauth/authorize (REQUIRED)
Token Endpoint:         https://oauth-openshift.apps.example.com/oauth/token (REQUIRED)
```

HTTP endpoints are rejected with clear error messages:
```
error: authorization endpoint validation failed: endpoint must use HTTPS, got http
```

**Why**: HTTP endpoints are vulnerable to DNS spoofing and credential interception. HTTPS ensures encrypted communication and server certificate validation.

### CSRF Protection (State Parameter)

OAuth state parameter prevents Cross-Site Request Forgery attacks in the SSO flow:

- **Generation**: 32 cryptographically random bytes, base64url-encoded (43 characters)
- **Flow**:
  1. Plugin generates unique state token
  2. Includes state in authorization URL
  3. OAuth server includes state in callback
  4. Plugin validates callback state matches original
- **Mismatch Detection**: Rejects callback if state doesn't match (possible CSRF attack)

### Timeout Protection

HTTP timeout set to **30 seconds** on token exchange requests:

- Prevents infinite hangs if OAuth server becomes unresponsive
- Uses separate HTTP client for token endpoint
- Timeout applies only to token exchange, not to browser authentication

### Token Type Validation

Only accepts `Bearer` token type from OAuth response:

```json
{
  "access_token": "sha256~...",
  "token_type": "Bearer",
  "expires_in": 86400
}
```

Rejects unsupported types with error:
```
error: unsupported token type: DPoP (expected Bearer)
```

## OAuth 2.0 / PKCE Implementation Details

### PKCE Flow (RFC 7636)

**Step 1: Generate PKCE Parameters**
```
code_verifier = base64url(32 random bytes)  // 43 characters
code_challenge = base64url(SHA256(code_verifier))
```

**Step 2: Authorization Request**
```
GET https://oauth-openshift.apps.example.com/oauth/authorize?
  client_id=openshift-cli-client
  &response_type=code
  &code_challenge=<BASE64URL(SHA256(verifier))>
  &code_challenge_method=S256
  &redirect_uri=http://127.0.0.1:<random>/callback
  &state=<random-state-token>
```

**Step 3: Token Exchange**
```
POST https://oauth-openshift.apps.example.com/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=<authorization-code>
&client_id=openshift-cli-client
&redirect_uri=http://127.0.0.1:<random>/callback
&code_verifier=<original-verifier>
```

### Callback Server

Local HTTP server receives OAuth callback:
- **Address**: `127.0.0.1:<random-port>` (loopback only, secure)
- **Path**: `/callback`
- **Validates**:
  - State parameter matches (CSRF protection)
  - Authorization code present
  - Error responses (OAuth error format)
- **Timeouts**:
  - Read: 10 seconds
  - Write: 10 seconds
  - Idle: 30 seconds
  - Max header: 4096 bytes

## Token Management

### Token Format

OpenShift tokens use format: `sha256~<hash>`

Example:
```
sha256~abcdefghijklmnopqrstuvwxyz1234567890abcdefghij
```

### Token Storage

Tokens stored as JSON in secure storage:

```json
{
  "token": "sha256~abcdefghijklmnopqrstuvwxyz1234567890abcdefghij",
  "expirationTimestamp": "2026-02-23T10:30:00Z"
}
```

### Token Validation

Two-phase validation ensures token freshness:

1. **Timestamp-Based Check** (Fast)
   - Parse ISO 8601 UTC timestamp
   - Compare with current time
   - Return immediately if not expired

2. **API-Based Check** (Fallback)
   - Call `/apis/user.openshift.io/v1/users/~`
   - Requires valid token in `Authorization: Bearer` header
   - Server responds with 200 (valid) or 401 (invalid)
   - Reuses HTTP connections

### Token Expiry

- **Default**: If `expires_in` not provided, default to 24 hours
- **Format**: ISO 8601 UTC: `2006-01-02T15:04:05Z`
- **Calculation**: `now + expires_in seconds`
- **Timezone**: Always UTC for consistency

OpenShift typically issues tokens with:
- **1-hour expiry** for regular sessions
- **24-hour default** if expires_in omitted
- **Configurable** via OAuth server settings

## HTTP Client Configuration

### Transport Settings

**TLS Configuration**:
```go
&tls.Config{
  InsecureSkipVerify: !verifySSL  // Use with caution in dev/test only
}
```

**Redirect Handling**:
```go
CheckRedirect: func(req *http.Request, via []*http.Request) error {
  return http.ErrUseLastResponse  // Don't follow redirects
}
```

Redirects are not followed because we need to extract tokens from `Location` header fragments.

### Connection Reuse

- Main HTTP client: Reused across requests for efficiency
- Token exchange: Separate client with 30-second timeout
- All response bodies read and discarded to enable connection reuse: `io.Copy(io.Discard, resp.Body)`

## Storage Backends

### macOS Keychain

- **Service**: Generic password item
- **Account**: `openshift-auth-plugin`
- **Item Names**: 
  - `{CLUSTER_NAME}-token` - cached access token
  - `{CLUSTER_NAME}-credentials` - stored username/password

### gopass

- **Path Structure**: `openshift-cli/{CLUSTER_NAME}/{key}`
  - `openshift-cli/my-cluster/token` - cached access token
  - `openshift-cli/my-cluster/credentials` - stored username/password
- **Format**: JSON stored as encrypted plain text

## Error Handling

### Graceful Degradation

Plugin attempts authentication in order, with clear error messages:

```
Method 1: Cached Token fails → Try Method 2
Method 2: SSO fails → Try Method 3
Method 3: Basic Auth fails → Output ExecCredential error, exit 1
```

### Error Output Format

All errors output as ExecCredential JSON to stderr:

```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "status": {
    "expirationTimestamp": "2006-01-02T15:04:05Z",
    "message": "error: specific error message here"
  }
}
```

## Logging

### Debug Output

All debug logs go to **stderr** (stdout reserved for ExecCredential JSON):

```
2026-02-23T08:30:15Z Attempting to use cached token...
2026-02-23T08:30:15Z Found cached token, validating...
2026-02-23T08:30:15Z Current timestamp: 2026-02-23T08:30:15Z
2026-02-23T08:30:15Z Expiry timestamp: 2026-02-23T09:30:15Z
2026-02-23T08:30:15Z Token is not expired by timestamp, considering it valid
```

### Timestamp Format

All timestamps use ISO 8601 UTC format: `2006-01-02T15:04:05Z`

## Browser Integration

### Supported Platforms

- **macOS**: `open` command
- **Linux**: `xdg-open` command
- **Windows**: `cmd /c start` command

### Fallback

If browser fails to open:
1. Plugin prints authorization URL to stderr
2. User can manually copy/paste URL to browser
3. Plugin still waits for callback via localhost

## Backward Compatibility

### Compatibility with Shell Version

Go version is 100% backward compatible:

- ✅ Same credential storage format
- ✅ Same environment variable names
- ✅ Same storage implementation
- ✅ Same token format and expiry
- ✅ Identical kubeconfig configuration

### Migration Path

Simply update kubeconfig path from shell script to Go binary - no credential migration needed.

## Best Practices

### For Operators

1. **Whitelist callback URL**: Add `http://127.0.0.1/*` to OAuth allowed redirect URLs
2. **Configure token expiry**: Balance security vs convenience (1 hour typical)
3. **Enable IdP integration**: Configure Azure AD, Okta, or other IdP
4. **Monitor OAuth logs**: Check for failed authentication attempts
5. **Use HTTPS**: Ensure OAuth endpoints have valid certificates

### For Users

1. **Use HTTPS**: Set `VERIFY_SSL=true` in production
2. **Store credentials securely**: Use Keychain or gopass
3. **Enable debug for troubleshooting**: Add `--debug` flag
4. **Check network connectivity**: Ensure access to OAuth endpoints
5. **Increase SSO timeout**: For slow networks, use `SSO_TIMEOUT=180`

### For Developers

1. **Always validate HTTPS**: Never skip this check
2. **Include state parameter**: Always use CSRF protection
3. **Set HTTP timeouts**: Prevent resource exhaustion
4. **Validate token types**: Only accept Bearer
5. **Test error paths**: Ensure graceful degradation

## Version History

### v1.1.0 (Current - Security Hardening Release)

**Security Improvements**:
- ✅ HTTPS validation for OAuth endpoints (prevents MITM)
- ✅ CSRF protection via state parameter (prevents CSRF)
- ✅ HTTP timeout on token exchange (prevents hangs)
- ✅ Token type validation (prevents type confusion)

**Test Coverage**: 114 tests, 70.5% coverage

### v1.0.0 (Initial Release)

- Three-method authentication flow
- PKCE support for SSO
- Keychain and gopass storage backends
- Shell script version
