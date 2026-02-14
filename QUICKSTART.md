# Quick Start Guide - Go Authentication Plugin

This guide will help you quickly set up and use the Go-based OpenShift authentication plugin.

## Prerequisites

### Install Go (if not already installed)

**macOS:**
```bash
brew install go
```

**Linux:**
```bash
# Download and install Go 1.21+
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Verify installation:
```bash
go version
```

## Installation

### Step 1: Build the Plugin

```bash
cd /path/to/openshift-tools

# Download dependencies
go mod download

# Build the binary
make build

# Or build manually
go build -o openshift-auth-plugin .
```

### Step 2: Install the Binary

```bash
# Install to /usr/local/bin (requires sudo)
sudo make install

# Or copy manually
sudo cp openshift-auth-plugin /usr/local/bin/
sudo chmod +x /usr/local/bin/openshift-auth-plugin
```

### Step 3: Verify Installation

```bash
which openshift-auth-plugin
# Should output: /usr/local/bin/openshift-auth-plugin
```

## Configuration

### Step 1: Update Your Kubeconfig

Edit `~/.kube/config` and add/modify your user configuration:

```yaml
users:
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: openshift-auth-plugin
      env:
      - name: CLUSTER_NAME
        value: my-cluster  # Change this to your cluster name
      - name: DEBUG
        value: "false"     # Set to "true" for debugging
      provideClusterInfo: true
      interactiveMode: IfAvailable
```

**Important**: 
- Replace `my-cluster` with a unique name for your cluster
- Set `provideClusterInfo: true` to automatically pass the server URL
- The `command` should be just `openshift-auth-plugin` (no path needed if installed in PATH)

### Step 2: Configure Your Context

Make sure your context uses the correct user:

```yaml
contexts:
- context:
    cluster: my-openshift-cluster
    user: my-openshift-user  # Must match the user name above
  name: my-context

current-context: my-context
```

## First Use

### Step 1: Test the Plugin

```bash
# Enable debug mode for first run
export DEBUG=true

# Try to access your cluster
kubectl get pods

# Or test any kubectl command
kubectl get nodes
```

### Step 2: Authenticate

On first run, the plugin will:

1. **Open your browser** to the OpenShift OAuth login page
2. **Prompt you to authenticate** with your SSO provider (Keycloak, Okta, Azure AD, etc.)
3. **Display a token** after successful authentication
4. **Ask you to copy the token** and paste it into the terminal

Example output:
```
==============================================================
OpenShift Authentication Required
==============================================================

Opening browser to: https://oauth-openshift.apps.cluster.example.com/oauth/authorize?...

Please complete authentication in your browser.
After authentication, you will receive an access token.

Enter access token: 
```

### Step 3: Paste the Token

1. Complete authentication in your browser
2. Copy the displayed token
3. Paste it into the terminal prompt
4. Press Enter

The plugin will:
- Validate the token
- Store it securely in macOS Keychain (or gopass)
- Output the credential to kubectl
- Cache it for future use (typically 24 hours)

## Subsequent Uses

After the first authentication, the plugin will:

1. **Check for cached token** in Keychain/gopass
2. **Validate the token** is still valid
3. **Use the cached token** if valid
4. **Re-authenticate** only if token is expired or invalid

You won't need to authenticate again until the token expires!

```bash
# These will use the cached token
kubectl get pods
kubectl get services
kubectl logs my-pod
```

## Troubleshooting

### Browser Doesn't Open

If the browser doesn't open automatically:

1. Look for the URL in the terminal output
2. Copy the URL manually
3. Paste it into your browser
4. Complete authentication
5. Copy the token and paste it back into the terminal

### Token Validation Fails

If you see "token validation failed":

```bash
# Delete the cached token
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"

# Try again
kubectl get pods
```

### SSL Certificate Errors

If you see SSL certificate errors:

```bash
# Option 1: Disable SSL verification (not recommended for production)
# Add to your kubeconfig env section:
- name: VERIFY_SSL
  value: "false"

# Option 2: Install the cluster's CA certificate
# Contact your cluster administrator
```

### Debug Mode

Enable debug logging to see what's happening:

```bash
# In your kubeconfig, set:
- name: DEBUG
  value: "true"

# Or export before running kubectl:
export DEBUG=true
kubectl get pods
```

### Check Stored Token

View your stored token (macOS):

```bash
# View token
security find-generic-password -a "openshift-auth-plugin" -s "my-cluster-token" -w

# Delete token
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"
```

## Advanced Configuration

### Using gopass Instead of Keychain

```yaml
env:
- name: SECRET_STORE
  value: "gopass"
- name: CLUSTER_NAME
  value: my-cluster
```

### Multiple Clusters

Configure different users for each cluster:

```yaml
users:
- name: cluster1-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: openshift-auth-plugin
      env:
      - name: CLUSTER_NAME
        value: cluster1
      provideClusterInfo: true

- name: cluster2-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: openshift-auth-plugin
      env:
      - name: CLUSTER_NAME
        value: cluster2
      provideClusterInfo: true
```

Each cluster will have its own cached token.

### Custom OAuth Client ID

If your cluster uses a custom OAuth client:

```yaml
env:
- name: SSO_CLIENT_ID
  value: "my-custom-client-id"
```

## Comparison with Shell Script

| Feature | Shell Script | Go Plugin |
|---------|-------------|-----------|
| Speed | ~500ms | ~50ms |
| Dependencies | curl, jq, base64 | None |
| Error Messages | Basic | Detailed |
| Cross-platform | Limited | Excellent |
| Maintenance | Harder | Easier |

## Next Steps

- Read the full [README-GO.md](README-GO.md) for detailed documentation
- Check [AGENTS-GO.md](AGENTS-GO.md) for development guidelines
- Report issues or contribute improvements

## Getting Help

1. **Enable debug mode**: Set `DEBUG=true`
2. **Check logs**: Look at stderr output
3. **Verify configuration**: Check your kubeconfig syntax
4. **Test manually**: Run `openshift-auth-plugin` directly with env vars
5. **Check storage**: Verify tokens are being stored correctly

## Common Commands

```bash
# Build
make build

# Install
sudo make install

# Test manually
export CLUSTER_NAME=my-cluster
export OPENSHIFT_URL=https://api.cluster.example.com:6443
export DEBUG=true
./openshift-auth-plugin

# View cached token (macOS)
security find-generic-password -a "openshift-auth-plugin" -s "my-cluster-token" -w

# Delete cached token (macOS)
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"

# Format code
make fmt

# Run tests
make test
```
