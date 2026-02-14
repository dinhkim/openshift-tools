# Agent Guidelines for OpenShift Auth Plugin (Go)

This document provides coding guidelines and context for AI coding agents working on the Go-based authentication plugin.

## Project Overview

**Type**: Go application  
**Purpose**: Kubernetes ExecCredential authentication plugin for OpenShift using OAuth/SSO  
**Language**: Go 1.21+  
**Architecture**: Modular package structure with clean separation of concerns

## Project Structure

```
.
├── main.go                      # Entry point, ExecCredential output
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── Makefile                     # Build automation
├── pkg/
│   ├── auth/
│   │   ├── auth.go             # Main authenticator logic
│   │   ├── client.go           # OpenShift API HTTP client
│   │   └── oauth.go            # OAuth flow (browser-based)
│   ├── config/
│   │   ├── config.go           # Configuration from env vars
│   │   └── logger.go           # Simple logging utility
│   └── storage/
│       ├── storage.go          # Storage interface
│       ├── keychain.go         # macOS Keychain backend
│       └── gopass.go           # gopass backend
└── README-GO.md                # User documentation
```

## Build, Test, and Lint Commands

### Building

```bash
# Build binary
make build

# Build and install to /usr/local/bin
make install

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -cover ./...

# Run specific package tests
go test -v ./pkg/auth
```

### Linting

```bash
# Run golangci-lint (requires installation)
make lint

# Format code
make fmt

# Run go vet
make vet
```

### Dependencies

```bash
# Download dependencies
make deps

# Or manually
go mod download
go mod tidy
```

## Code Style Guidelines

### General Go Standards

1. **Follow Effective Go**: https://go.dev/doc/effective_go
2. **Use gofmt**: All code must be formatted with `gofmt`
3. **Package naming**: Short, lowercase, single-word names (e.g., `auth`, `storage`, `config`)
4. **Exported vs Unexported**:
   - Exported (public): `PascalCase` - starts with uppercase
   - Unexported (private): `camelCase` - starts with lowercase

### Naming Conventions

1. **Variables**:
   - Local: `camelCase` (e.g., `tokenData`, `serverURL`)
   - Package-level: `camelCase` for unexported, `PascalCase` for exported
   - Constants: `PascalCase` or `UPPER_SNAKE_CASE` for package-level

2. **Functions/Methods**:
   - Exported: `PascalCase` (e.g., `GetToken`, `ValidateToken`)
   - Unexported: `camelCase` (e.g., `tryStoredToken`, `buildAuthorizationURL`)

3. **Types/Structs**:
   - Always `PascalCase` (e.g., `Authenticator`, `OAuthInfo`)
   - Interfaces: Often end with `-er` (e.g., `Storage`, `Logger`)

4. **Files**:
   - Lowercase with underscores if needed (e.g., `auth.go`, `oauth.go`)
   - Test files: `*_test.go`

### Error Handling

1. **Always check errors**:
   ```go
   result, err := someFunction()
   if err != nil {
       return fmt.Errorf("context: %w", err)
   }
   ```

2. **Wrap errors with context**:
   ```go
   return fmt.Errorf("failed to authenticate: %w", err)
   ```

3. **Don't panic**: Use error returns instead (except for truly unrecoverable situations)

4. **ExecCredential errors**: Must output JSON to stderr and exit with non-zero code

### Struct Design

1. **Composition over inheritance**:
   ```go
   type Authenticator struct {
       config  *config.Config
       storage storage.Storage
       logger  *config.Logger
       client  *OpenshiftClient
   }
   ```

2. **Constructor pattern**:
   ```go
   func NewAuthenticator(cfg *config.Config, store storage.Storage, logger *config.Logger) *Authenticator {
       return &Authenticator{
           config:  cfg,
           storage: store,
           logger:  logger,
           client:  NewOpenshiftClient(cfg.ServerURL, cfg.VerifySSL, logger),
       }
   }
   ```

3. **JSON tags for serialization**:
   ```go
   type TokenData struct {
       Token               string    `json:"token"`
       ExpirationTimestamp time.Time `json:"expirationTimestamp"`
   }
   ```

### Interface Design

1. **Keep interfaces small**:
   ```go
   type Storage interface {
       GetToken() (*TokenData, error)
       StoreToken(token string, expiry time.Time) error
   }
   ```

2. **Accept interfaces, return structs**:
   ```go
   func NewAuthenticator(cfg *config.Config, store storage.Storage, logger *config.Logger) *Authenticator
   ```

### HTTP Client Patterns

1. **Reuse HTTP clients**:
   ```go
   type OpenshiftClient struct {
       httpClient *http.Client
   }
   ```

2. **Set timeouts**:
   ```go
   httpClient: &http.Client{
       Timeout: 30 * time.Second,
   }
   ```

3. **Handle TLS configuration**:
   ```go
   transport := &http.Transport{
       TLSClientConfig: &tls.Config{
           InsecureSkipVerify: !verifySSL,
       },
   }
   ```

4. **Always close response bodies**:
   ```go
   resp, err := client.Get(url)
   if err != nil {
       return err
   }
   defer resp.Body.Close()
   ```

### Logging

1. **Use structured logging** (via our Logger):
   ```go
   logger.Debug("Validating token with: %s", url)
   logger.Error("Failed to store token: %v", err)
   ```

2. **Log to stderr**: Never log to stdout (reserved for ExecCredential JSON)

3. **Debug logs**: Controlled by `DEBUG` environment variable

### Time Handling

1. **Use time.Time**: Not strings for timestamps
2. **UTC for storage**: Always store times in UTC
3. **Format for JSON**: Use RFC3339 or custom format
   ```go
   expiry := time.Now().Add(24 * time.Hour)
   ```

### Command Execution

1. **Use exec.Command**:
   ```go
   cmd := exec.Command("security", "find-generic-password", "-w")
   output, err := cmd.Output()
   ```

2. **Check command availability**:
   ```go
   if _, err := exec.LookPath("gopass"); err != nil {
       return fmt.Errorf("gopass not found in PATH")
   }
   ```

### JSON Handling

1. **Use encoding/json**:
   ```go
   var data TokenData
   if err := json.Unmarshal(bytes, &data); err != nil {
       return err
   }
   ```

2. **Pretty print for output**:
   ```go
   output, err := json.MarshalIndent(cred, "", "  ")
   ```

### Comments and Documentation

1. **Package comments**: Every package should have a doc comment
   ```go
   // Package auth handles OpenShift OAuth authentication
   package auth
   ```

2. **Exported symbols**: Must have doc comments
   ```go
   // GetToken retrieves a valid token using various authentication methods
   func (a *Authenticator) GetToken() (string, time.Time, error)
   ```

3. **Comment format**: Start with the name of the thing being described
   ```go
   // Authenticator handles OpenShift authentication
   type Authenticator struct { ... }
   ```

## Project-Specific Patterns

### ExecCredential Output Format

The plugin MUST output valid Kubernetes ExecCredential JSON to stdout:

```go
cred := &clientauthv1.ExecCredential{
    TypeMeta: metav1.TypeMeta{
        APIVersion: "client.authentication.k8s.io/v1",
        Kind:       "ExecCredential",
    },
    Status: &clientauthv1.ExecCredentialStatus{
        Token:               token,
        ExpirationTimestamp: &metav1.Time{Time: expiry},
    },
}
```

### Authentication Flow

1. Check cached token → validate → return if valid
2. Attempt SSO authentication → cache → return
3. Return error if all methods fail

### Storage Abstraction

- Define `Storage` interface in `pkg/storage/storage.go`
- Implement backends: `KeychainStorage`, `GopassStorage`
- Factory pattern: `NewStorage(storeType, clusterName, logger)`

### OAuth Flow

1. Fetch OAuth metadata from `/.well-known/oauth-authorization-server`
2. Build authorization URL with proper parameters
3. Open browser to authorization endpoint
4. Prompt user for token from terminal
5. Cache token for future use

## Dependencies

### Core Dependencies

- `k8s.io/client-go`: Kubernetes client library (ExecCredential types)
- `k8s.io/apimachinery`: Kubernetes API machinery (metav1 types)
- `golang.org/x/oauth2`: OAuth 2.0 support (future use)

### Standard Library

- `encoding/json`: JSON marshaling/unmarshaling
- `net/http`: HTTP client
- `crypto/tls`: TLS configuration
- `os/exec`: Execute external commands
- `time`: Time handling

## Testing Guidelines

### Unit Tests

1. **Test file naming**: `*_test.go`
2. **Test function naming**: `TestFunctionName`
3. **Table-driven tests**:
   ```go
   func TestValidateToken(t *testing.T) {
       tests := []struct {
           name    string
           token   string
           wantErr bool
       }{
           {"valid token", "valid-token", false},
           {"invalid token", "invalid", true},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // test logic
           })
       }
   }
   ```

### Integration Tests

- Test with real OpenShift clusters (optional)
- Mock HTTP responses for OAuth endpoints
- Test storage backends with temporary credentials

## Security Considerations

1. **Never log tokens**: Except in DEBUG mode, be very careful
2. **Secure storage**: Use Keychain or gopass, never plain files
3. **TLS verification**: Support both verified and unverified (with warning)
4. **Input validation**: Validate all user input and environment variables
5. **Error messages**: Don't leak sensitive information in errors

## Common Tasks

### Adding a New Storage Backend

1. Create new file in `pkg/storage/` (e.g., `vault.go`)
2. Implement `Storage` interface
3. Add to `NewStorage` factory function
4. Add tests
5. Update documentation

### Adding New Authentication Method

1. Add method to `Authenticator` in `pkg/auth/auth.go`
2. Call from `GetToken()` in appropriate order
3. Implement token caching
4. Add tests
5. Update documentation

### Modifying OAuth Flow

1. Edit `pkg/auth/oauth.go`
2. Update `buildAuthorizationURL()` for new parameters
3. Modify `Authenticate()` for new flow steps
4. Test with real OAuth provider
5. Update documentation

## Best Practices

1. **Keep packages focused**: Each package has a single responsibility
2. **Minimize dependencies**: Only add dependencies when necessary
3. **Error context**: Always wrap errors with context
4. **Fail fast**: Validate configuration early
5. **Graceful degradation**: Fall back to alternative methods when possible
6. **Cross-platform**: Test on macOS, Linux, and Windows (WSL)
7. **Backward compatibility**: Don't break existing kubeconfig configurations

## Performance Considerations

1. **Reuse HTTP clients**: Don't create new clients for each request
2. **Cache tokens**: Avoid unnecessary authentication
3. **Lazy initialization**: Only initialize what's needed
4. **Timeout handling**: Set reasonable timeouts for HTTP requests
5. **Binary size**: Keep dependencies minimal for smaller binary

## Debugging

1. **Enable debug logging**: `export DEBUG=true`
2. **Test manually**: Run binary directly with env vars set
3. **Check kubeconfig**: Verify exec configuration is correct
4. **Validate JSON output**: Pipe output through `jq`
5. **Check storage**: Verify tokens are stored correctly

## Migration from Shell Script

Key improvements over shell script:
- **Performance**: Native binary vs spawning processes
- **Error handling**: Comprehensive error types and wrapping
- **Type safety**: Compile-time checks vs runtime errors
- **Testing**: Unit tests vs manual testing only
- **Maintainability**: Modular code vs monolithic script
- **Cross-platform**: Better Windows/Linux support
