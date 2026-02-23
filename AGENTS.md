# AGENTS.md — OpenShift Auth Plugin

## Project Overview

Kubernetes ExecCredential authentication plugin for OpenShift clusters. Implements `client.authentication.k8s.io/v1beta1`. When configured in kubeconfig, it authenticates `kubectl` commands automatically.

**Available Implementations:**
- **Go version** (recommended): Self-contained binary, better performance, comprehensive error handling
- **Shell script version** (legacy): Bash script with external dependencies

## Build / Lint / Test Commands

### Go Version (Recommended)

```bash
# Build binary
make build

# Run tests
make test

# Generate coverage report (creates coverage.html)
make coverage

# Lint (requires golangci-lint)
make lint

# Format code
make fmt

# Run go vet
make vet

# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean

# Manual test with debug output
DEBUG=true CLUSTER_NAME=test-cluster OPENSHIFT_URL=https://api.cluster.example.com:6443 ./bin/openshift-auth-plugin

# Test with flags
./bin/openshift-auth-plugin --cluster-name test-cluster --openshift-url https://api.cluster.example.com:6443 --debug
```

### Shell Script (Legacy)

```bash
# Syntax check
bash -n openshift-auth-plugin.sh

# Lint with shellcheck
shellcheck openshift-auth-plugin.sh

# Manual test with debug output
DEBUG=true CLUSTER_NAME=test-cluster OPENSHIFT_URL=https://api.cluster.example.com:6443 ./openshift-auth-plugin.sh

# Validate JSON output
./openshift-auth-plugin.sh 2>/dev/null | jq .
```

No automated test framework exists for the shell script.

## Code Style — Go

### Project Structure
```
cmd/openshift-auth-plugin/    # Entry point & main function
internal/                      # Private packages
  auth/                        # OAuth & token validation logic
  config/                      # Configuration loading
  credential/                  # ExecCredential types & output
  log/                         # Debug logger
  storage/                     # Storage interface & implementations
```

### Go Best Practices
- Use standard package layout (`cmd/`, `internal/`)
- Interface-based design for testability
- Proper error handling with `error` returns
- Clear separation of concerns (each package has single responsibility)
- No global mutable state
- Reuse HTTP client with custom transport
- Always defer close for resources
- Use context for cancellation when needed

### Naming Conventions
- Package names: lowercase, single word (e.g., `auth`, `storage`)
- Exported types/functions: PascalCase (e.g., `NewAuthenticator`, `Storage`)
- Unexported types/functions: camelCase (e.g., `extractTokenFromFragment`)
- Interfaces: noun without `-er` suffix for complex interfaces, with `-er` for simple (e.g., `Storage`)
- Constants: PascalCase or UPPER_SNAKE_CASE for package-level

### Error Handling
- Return errors, don't panic (except for unrecoverable initialization errors)
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Check errors immediately after operations
- Use `credential.OutputError()` for fatal errors (outputs JSON to stderr, exits 1)
- Log non-fatal errors with logger if debug enabled

### HTTP & Network
- Use custom `http.Transport` for TLS configuration
- Set `CheckRedirect` to prevent following redirects (need Location header for OAuth)
- Always defer `resp.Body.Close()` after HTTP requests
- Read and discard response body for connection reuse: `io.Copy(io.Discard, resp.Body)`
- Set appropriate headers (`Authorization`, `Accept`, `X-CSRF-Token`)

### JSON Handling
- Use struct tags for proper JSON marshaling: `json:"fieldName,omitempty"`
- Always validate JSON before unmarshaling
- Use `json.Encoder` with `SetIndent` for pretty output
- Write ExecCredential JSON exclusively to stdout; errors to stderr

### Logging
- Use structured logging with logger package
- ISO 8601 UTC timestamps: `time.Now().UTC().Format("2006-01-02T15:04:05Z")`
- All logs go to stderr, never stdout
- Controlled by `DEBUG` env var or `--debug` flag
- Never log sensitive data (tokens, passwords)

### Testing
- Use table-driven tests for comprehensive coverage
- Test file naming: `*_test.go`
- Mock interfaces for unit testing (see `storage.MockStorage`)
- Use `t.Helper()` in helper functions
- Use `httptest` for testing HTTP interactions
- Separate unit tests from integration tests with build tags

**Current Test Coverage:**
- `internal/log`: 100% coverage
- `internal/config`: 97.2% coverage
- `internal/credential`: 88.9% coverage
- `internal/auth`: 70.7% coverage
- `internal/storage`: 59.3% coverage (mock: 100%, real backends: 0% - require system integration)

Note: `internal/auth` coverage is lower due to SSO browser-dependent code paths (browser open, full SSO end-to-end) that are not unit-testable. PKCE generation, token exchange, and callback server are fully tested.

Note: Real storage backends (keychain, gopass) are not unit tested as they require system commands. They are tested via integration/manual testing.

## Code Style — Shell Script

### Safety and Structure
- Shebang: `#!/bin/bash` with `set -euo pipefail` immediately after
- All output to stdout must be valid ExecCredential JSON; everything else goes to stderr
- Use `local` for every variable inside functions

### Naming
- Environment/global variables: `UPPER_SNAKE_CASE` (`OPENSHIFT_URL`, `VERIFY_SSL`)
- Local variables: `lower_snake_case` (`server_url`, `access_token`)
- Public functions: `lower_snake_case` (`get_oauth_info`, `validate_token`)
- Private/helper functions: prefix with `_` (`_get_secret`, `_store_secret`)

### Error Handling
- Fatal errors: call `error_exit "message"` which outputs ExecCredential JSON to stderr and exits 1
- Non-fatal: `return 1`
- Check command success with: `if ! result=$(command); then error_exit "msg"; fi`
- Always validate JSON: `echo "$response" | jq . >/dev/null 2>&1`

### Conditionals and Strings
- Always use `[[ ]]` (not `[ ]`), quote all variable expansions: `"$var"`
- Default values: `${VAR:-default}`, null checks: `[[ -z "$var" ]]` / `[[ -n "$var" ]]`
- Check command exists: `command -v "$cmd" >/dev/null 2>&1`

### HTTP / curl
- Use `$CURL_OPTS` for SSL config (`-k` when `VERIFY_SSL=false`)
- Always `-s` (silent) and `2>/dev/null`
- Capture HTTP status: `curl -w "%{http_code}"`, extract with `${response: -3}`

### Logging
- Use `log()` function, outputs to stderr, controlled by `DEBUG` env var
- ISO 8601 UTC timestamps: `date -u +'%Y-%m-%dT%H:%M:%SZ'`
- Use heredocs for multi-line JSON output (`cat << EOF ... EOF`)

## Security Rules
- Never log tokens or credentials (even in DEBUG mode, be cautious)
- Use secure storage only (macOS Keychain or gopass), never plain files
- Encode credentials properly for HTTP Basic Auth (`base64 -w 0`)
- Don't leak sensitive info in error messages
- Stdout is exclusively for ExecCredential JSON

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | Yes | — | Cluster identifier for credential storage |
| `OPENSHIFT_URL` | Conditional | — | API URL (or use `provideClusterInfo`) |
| `VERIFY_SSL` | No | `false` | SSL certificate verification |
| `SECRET_STORE` | No | `keychain` | `keychain` or `gopass` |
| `DEBUG` | No | `false` | Debug logging to stderr |
| `SSO_TIMEOUT` | No | `120` | Seconds to wait for SSO browser authentication |

## Commit Message Style
- Prefix with type: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`
- Lowercase after prefix
- Example: `feat: improve the token validation process`

## Authentication Flow

### Go Version (3-method flow)
1. Check cached token in secure storage → validate (check expiry + API call) → use if valid
2. SSO/PKCE authentication → open browser → user authenticates via IdP → exchange code for token → cache → output ExecCredential JSON
3. Fall back to username/password basic auth → cache new token → output ExecCredential JSON
4. On failure, output ExecCredential error JSON to stderr, exit 1

### Shell Script (2-method flow)
1. Check cached token in secure storage → validate (check expiry + API call) → use if valid
2. Fall back to username/password basic auth → cache new token → output ExecCredential JSON
3. On failure, output ExecCredential error JSON to stderr, exit 1

### SSO/PKCE Flow Details (Go Version Only)
The SSO flow uses OAuth 2.0 Authorization Code + PKCE (RFC 7636) with `client_id=openshift-cli-client`:
1. Fetch OAuth metadata from `/.well-known/oauth-authorization-server` (get `authorization_endpoint` and `token_endpoint`)
2. Generate PKCE parameters: `code_verifier` (32 random bytes, base64url) and `code_challenge` (SHA-256 of verifier, base64url)
3. Start a local HTTP callback server on `127.0.0.1:<random-port>/callback`
4. Open browser to authorization URL with PKCE params (`response_type=code`, `code_challenge_method=S256`)
5. User authenticates via SSO (Azure AD, Okta, etc.) in browser
6. OAuth server redirects to local callback with authorization code
7. Exchange code for access token via POST to `token_endpoint` with `code_verifier`
8. Store token, output ExecCredential JSON, shut down callback server
