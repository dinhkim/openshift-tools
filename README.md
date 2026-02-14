# OpenShift Tools

This repository contains a collection of tools for working with OpenShift.

## OpenShift Auth Plugin

A Kubernetes ExecCredential authentication plugin for OpenShift clusters. Available in both Go and Shell implementations.

### Go Version (Recommended)

The Go version is a self-contained binary with better performance and error handling.

#### Building

```bash
# Build the binary
make build

# Install to $GOPATH/bin
make install
```

#### Prerequisites

- Go 1.21 or later (for building)
- `gopass` (optional): If you want to use `gopass` to store your credentials.
- macOS Keychain access (default storage backend on macOS)

#### Configuration

The plugin can be configured via environment variables or command-line flags (flags take precedence):

| Environment Variable / Flag | Description | Default |
| --- | --- | --- |
| `CLUSTER_NAME` / `--cluster-name` | Cluster identifier for credential storage (required) | |
| `OPENSHIFT_URL` / `--openshift-url` | OpenShift API URL | |
| `VERIFY_SSL` / `--verify-ssl` | Verify SSL certificates | `false` |
| `SECRET_STORE` / `--secret-store` | Secret store backend: `keychain` or `gopass` | `keychain` |
| `DEBUG` / `--debug` | Enable debug logging to stderr | `false` |

#### kubeconfig Setup

```yaml
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: /path/to/openshift-auth-plugin
      env:
        - name: "CLUSTER_NAME"
          value: "my-openshift-cluster"
      provideClusterInfo: true
```

Replace `/path/to/openshift-auth-plugin` with the actual path to the binary, and `my-openshift-cluster` with your cluster name.

#### Storing Credentials

**macOS Keychain:**

```bash
security add-generic-password -a "openshift-auth-plugin" -s "my-openshift-cluster-credentials" -l "my-openshift-cluster credentials" -w '{ "username": "my-username", "password": "my-password" }'
```

**gopass:**

```bash
# Store credentials (will prompt for JSON input)
gopass insert -f my-openshift-cluster credentials
# Enter: { "username": "my-username", "password": "my-password" }
```

Replace `my-openshift-cluster` with your `CLUSTER_NAME`.

#### Running & Testing

```bash
# Build and run
make build
./bin/openshift-auth-plugin --cluster-name my-cluster --debug

# Or with environment variables
export CLUSTER_NAME=my-cluster
export DEBUG=true
./bin/openshift-auth-plugin

# Test help output
./bin/openshift-auth-plugin --help
```

---

## openshift-auth-plugin.sh (Shell Version)

The legacy shell script version of the authentication plugin.

### Prerequisites (Shell Version)

Before using this script, you need to have the following tools installed:

- `curl`: For making HTTP requests to the OpenShift cluster.
- `jq`: For parsing JSON responses from the OpenShift API.
- `base64`: For encoding credentials.
- `date`: For handling token expiration timestamps.
- `gopass` (optional): If you want to use `gopass` to store your credentials.
- `security` (macOS only): If you want to use the macOS Keychain to store your credentials.

### How it Works (Shell Version)

The script can be configured to authenticate in two ways:

1.  **Token Authentication**: If a valid, unexpired token is found in the configured secret store (Keychain or `gopass`), it will be used for authentication.
2.  **Username/Password Authentication**: If a token is not available or has expired, the script will try to authenticate using a username and password from the secret store.

Upon successful authentication, the script provides a valid token to `kubectl`.

### Configuration (Shell Version)

The shell script is configured through environment variables. These can be set in your `~/.bash_profile`, `~/.zshrc`, or a similar shell configuration file.

| Environment Variable | Description | Default |
| --- | --- | --- |
| `OPENSHIFT_URL` | The URL of your OpenShift cluster. | |
| `OPENSHIFT_USERNAME` | Your OpenShift username. | |
| `OPENSHIFT_PASSWORD` | Your OpenShift password. | |
| `CLUSTER_NAME` | The name of cluster you are using. | |
| `VERIFY_SSL` | Whether to verify SSL certificates. | `false` |
| `SECRET_STORE` | The secret store to use. Can be `keychain` or `gopass`. | `keychain` |

### kubeconfig Setup (Shell Version)

To use the shell script as an exec plugin, modify your `~/.kube/config` file:

```yaml
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: /path/to/openshift-auth-plugin.sh
      env:
        - name: "CLUSTER_NAME"
          value: "my-openshift-cluster"
      provideClusterInfo: true
```

Replace `/path/to/openshift-auth-plugin.sh` with the actual path to the script, and `my-openshift-cluster` with the name of your cluster.

---

## How it Works

Both implementations use the same authentication flow:

1. **Token Authentication**: If a valid, unexpired token is found in the configured secret store (Keychain or `gopass`), it will be used for authentication.
2. **Username/Password Authentication**: If a token is not available or has expired, the plugin will authenticate using stored username and password credentials.

Upon successful authentication, the plugin outputs a valid ExecCredential JSON to kubectl with the token and expiration timestamp.

---

## Migration from Shell to Go

The Go version is **100% backward compatible** with the shell version. To migrate:

1. **Build the Go binary**:
   ```bash
   make build
   # Or install to $GOPATH/bin
   make install
   ```

2. **Update your kubeconfig**: Simply change the `command:` path:
   ```yaml
   # Before
   command: /path/to/openshift-auth-plugin.sh
   
   # After
   command: /path/to/openshift-auth-plugin
   # Or if installed to $GOPATH/bin and in your PATH
   command: openshift-auth-plugin
   ```

3. **No other changes needed**: 
   - Existing credentials in Keychain/gopass work as-is
   - Environment variables work the same way
   - Secret storage key naming is identical
   - Token format is unchanged

4. **Optional**: Take advantage of new features:
   ```yaml
   # Use command-line flags instead of env vars
   command: /path/to/openshift-auth-plugin
   args:
     - --cluster-name=my-cluster
     - --debug
   provideClusterInfo: true
   ```

## Troubleshooting

### Debug Mode

Enable debug logging to see detailed execution flow:

```bash
# Go version
./bin/openshift-auth-plugin --cluster-name my-cluster --debug

# Shell version
DEBUG=true ./openshift-auth-plugin.sh
```

Debug output goes to stderr and includes:
- Configuration values
- OAuth endpoint discovery
- Token validation checks
- HTTP request/response details
- Timestamp comparisons

### Common Issues

**"CLUSTER_NAME is required"**
- Set the `CLUSTER_NAME` environment variable or `--cluster-name` flag

**"No valid authentication method found"**
- Credentials are not stored in the secret store
- Follow the "Storing Credentials" section to save your username/password

**"Failed to connect to OAuth endpoint"**
- Check `OPENSHIFT_URL` is correct
- If using `provideClusterInfo: true` in kubeconfig, ensure your cluster context is set correctly
- Check network connectivity to the OpenShift cluster

**"Token validation failed"**
- Your cached token has expired and will be refreshed automatically
- If credentials are not stored, authentication will fail

**SSL certificate errors**
- For development/test clusters with self-signed certs, use `VERIFY_SSL=false` or `--verify-ssl=false`
- For production, ensure your system trusts the cluster's CA certificate

## Development

### Project Structure

```
openshift-tools/
├── cmd/openshift-auth-plugin/    # Main entry point
│   └── main.go
├── internal/                      # Internal packages
│   ├── auth/                      # Authentication logic
│   │   ├── oauth.go              # OAuth flow
│   │   └── token.go              # Token validation
│   ├── config/                    # Configuration
│   │   └── config.go
│   ├── credential/                # ExecCredential types
│   │   └── execcredential.go
│   ├── log/                       # Logging
│   │   └── log.go
│   └── storage/                   # Credential storage
│       ├── storage.go            # Interface
│       ├── keychain.go           # macOS Keychain
│       └── gopass.go             # gopass
├── Makefile                       # Build automation
├── go.mod                         # Go module
└── README.md                      # This file
```

### Building & Testing

```bash
# Format code
make fmt

# Lint (requires golangci-lint)
make lint

# Run tests
make test

# Build
make build

# Clean
make clean
```

## License

This project is open source. See repository for license details.
