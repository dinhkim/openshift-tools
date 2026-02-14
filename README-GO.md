# OpenShift Authentication Plugin (Go)

A Go-based Kubernetes authentication plugin for OpenShift clusters that implements the same OAuth login flow as the `oc` CLI.

## Features

- **OAuth/SSO Authentication**: Browser-based authentication using OpenShift's OAuth server
- **Token Caching**: Securely stores tokens in macOS Keychain or gopass
- **Token Validation**: Automatically validates and refreshes expired tokens
- **ExecCredential API**: Implements Kubernetes client-go credential plugin specification
- **Cross-platform**: Works on macOS, Linux, and Windows (WSL)

## Prerequisites

- Go 1.21 or later (for building)
- `jq` (for JSON parsing in some scenarios)
- One of the following for secure token storage:
  - macOS Keychain (default on macOS)
  - `gopass` (cross-platform password manager)

## Installation

### Build from source

```bash
# Clone the repository
cd /path/to/openshift-tools

# Download dependencies
make deps

# Build the binary
make build

# Install to /usr/local/bin
make install
```

### Manual installation

```bash
go build -o openshift-auth-plugin .
sudo install -m 755 openshift-auth-plugin /usr/local/bin/
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENSHIFT_URL` | OpenShift API server URL | (required) |
| `CLUSTER_NAME` | Cluster identifier for credential storage | (required) |
| `VERIFY_SSL` | Verify SSL certificates | `false` |
| `SECRET_STORE` | Storage backend: `keychain` or `gopass` | `keychain` |
| `DEBUG` | Enable debug logging | `false` |
| `SSO_ENABLED` | Enable SSO authentication | `true` |
| `SSO_CLIENT_ID` | OAuth client ID | `openshift-browser-client` |
| `SSO_PROVIDER` | SSO provider name (for display) | (optional) |

### Kubeconfig Setup

Add the following to your `~/.kube/config`:

```yaml
users:
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: openshift-auth-plugin
      env:
      - name: CLUSTER_NAME
        value: my-openshift-cluster
      - name: DEBUG
        value: "false"
      provideClusterInfo: true
      interactiveMode: IfAvailable
```

**Important**: Set `provideClusterInfo: true` to automatically provide the server URL, or set `OPENSHIFT_URL` explicitly in the `env` section.

## Usage

Once configured in your kubeconfig, the plugin will automatically authenticate when you use `kubectl`:

```bash
# The plugin will automatically run when needed
kubectl get pods

# First time: Opens browser for SSO authentication
# Subsequent times: Uses cached token
```

### Manual Testing

You can test the plugin directly:

```bash
# Set required environment variables
export CLUSTER_NAME=my-cluster
export OPENSHIFT_URL=https://api.my-cluster.example.com:6443
export DEBUG=true

# Run the plugin
./openshift-auth-plugin
```

## Authentication Flow

1. **Check Cached Token**: Retrieves token from storage (Keychain/gopass)
2. **Validate Token**: Checks if token is valid and not expired
3. **SSO Authentication** (if needed):
   - Fetches OAuth metadata from OpenShift
   - Opens browser to OAuth authorization endpoint
   - User authenticates with SSO provider (Keycloak, Okta, Azure AD, etc.)
   - User copies the displayed token
   - Token is cached for future use

## Storage Backends

### macOS Keychain (default)

Tokens are stored in the macOS Keychain with:
- Account: `openshift-auth-plugin`
- Service: `<cluster-name>-token`

View stored tokens:
```bash
security find-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"
```

Delete stored token:
```bash
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"
```

### gopass

Tokens are stored in gopass at:
- Path: `<cluster-name>/token`

View stored token:
```bash
gopass show my-cluster/token
```

Delete stored token:
```bash
gopass rm my-cluster/token
```

## Development

### Project Structure

```
.
├── main.go                 # Entry point
├── pkg/
│   ├── auth/
│   │   ├── auth.go        # Main authenticator
│   │   ├── client.go      # OpenShift API client
│   │   └── oauth.go       # OAuth flow handler
│   ├── config/
│   │   ├── config.go      # Configuration management
│   │   └── logger.go      # Logging utilities
│   └── storage/
│       ├── storage.go     # Storage interface
│       ├── keychain.go    # macOS Keychain backend
│       └── gopass.go      # gopass backend
├── go.mod
├── go.sum
├── Makefile
└── README-GO.md
```

### Building

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

### Adding New Storage Backends

1. Implement the `Storage` interface in `pkg/storage/storage.go`
2. Add your backend to the `NewStorage` factory function
3. Update documentation

## Comparison with Shell Script

| Feature | Shell Script | Go Plugin |
|---------|-------------|-----------|
| Performance | Slower (spawns processes) | Faster (native binary) |
| Dependencies | curl, jq, base64, date | None (statically linked) |
| Error Handling | Basic | Comprehensive |
| Code Maintainability | Harder to test/debug | Easier to test/debug |
| Cross-platform | Limited | Better support |
| Type Safety | None | Full type safety |

## Troubleshooting

### Enable Debug Logging

```bash
export DEBUG=true
kubectl get pods
```

### Token Validation Fails

```bash
# Delete cached token and re-authenticate
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"
kubectl get pods
```

### SSL Certificate Issues

```bash
# Disable SSL verification (not recommended for production)
export VERIFY_SSL=false
```

### Browser Doesn't Open

The plugin will print the OAuth URL. Copy and paste it into your browser manually.

## Security Considerations

- Tokens are stored securely in Keychain or gopass
- SSL verification is disabled by default (set `VERIFY_SSL=true` for production)
- Debug mode may log sensitive information - use only for troubleshooting
- Tokens are validated before use

## License

MIT

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request
