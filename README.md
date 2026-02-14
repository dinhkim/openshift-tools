# OpenShift Tools

This repository contains authentication plugins for working with OpenShift clusters.

## Available Implementations

### 🚀 **Go Plugin** (Recommended)
A high-performance, native Go implementation with better error handling and cross-platform support.

- **File**: `openshift-auth-plugin` (binary)
- **Documentation**: [README-GO.md](README-GO.md)
- **Quick Start**: [QUICKSTART.md](QUICKSTART.md)
- **Performance**: ~50ms execution time
- **Dependencies**: None (statically linked)

### 📜 **Shell Script** (Legacy)
The original bash implementation, still functional but slower.

- **File**: `openshift-auth-plugin.sh`
- **Documentation**: See below
- **Performance**: ~500ms execution time
- **Dependencies**: curl, jq, base64, date

---

## openshift-auth-plugin.sh (Shell Script)

This script is a `kubeconfig` exec plugin for handling OpenShift authentication. It allows you to authenticate with an OpenShift cluster using either a token or a username and password. The script can securely store your credentials in the macOS Keychain or `gopass`.

### Prerequisites

Before using this script, you need to have the following tools installed:

- `curl`: For making HTTP requests to the OpenShift cluster.
- `jq`: For parsing JSON responses from the OpenShift API.
- `base64`: For encoding credentials.
- `date`: For handling token expiration timestamps.
- `gopass` (optional): If you want to use `gopass` to store your credentials.
- `security` (macOS only): If you want to use the macOS Keychain to store your credentials.

### How it Works

The script can be configured to authenticate in three ways (tried in order):

1.  **Token Authentication**: If a valid, unexpired token is found in the configured secret store (Keychain or `gopass`), it will be used for authentication.
2.  **SSO Authentication** (New): If no valid cached token exists, the script will attempt browser-based SSO authentication via OpenShift's OAuth server with your configured identity provider (Keycloak, Okta, Azure AD, etc.).
3.  **Username/Password Authentication**: If SSO is disabled or unavailable, the script will try to authenticate using a username and password from the secret store.

Upon successful authentication, the script provides a valid token to `kubectl`.

### Configuration

The script is configured through environment variables. These can be set in your `~/.bash_profile`, `~/.zshrc`, or a similar shell configuration file.

| Environment Variable | Description | Default |
| --- | --- | --- |
| `OPENSHIFT_URL` | The URL of your OpenShift cluster. | |
| `OPENSHIFT_USERNAME` | Your OpenShift username. | |
| `OPENSHIFT_PASSWORD` | Your OpenShift password. | |
| `CLUSTER_NAME` | The name of cluster you are using. | |
| `VERIFY_SSL` | Whether to verify SSL certificates. | `false` |
| `SECRET_STORE` | The secret store to use. Can be `keychain` or `gopass`. | `keychain` |
| `SSO_ENABLED` | Enable SSO (browser-based) authentication. | `true` |
| `SSO_PROVIDER` | Name of your SSO provider (for display purposes). | |
| `SSO_CLIENT_ID` | OAuth client ID for browser-based authentication. | `openshift-browser-client` |

### `kubeconfig` Setup

To use this script as an exec plugin, you need to modify your `~/.kube/config` file. Add a `user` entry like this:

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

### Storing Credentials

You can store your OpenShift username and password in your chosen secret store.

#### macOS Keychain

```bash
# Store your credentails
security add-generic-password -a "openshift-auth-plugin" -s "my-openshift-cluster-credentials" -l "my-openshift-cluster credentials" -T "" -w "{ \"username\": \"my-username\", \"password\": \"my-password\" }"

```

#### gopass

```bash
# Store your credentails
gopass insert -f my-openshift-cluster credentials

```

Replace `my-openshift-cluster` with your `CLUSTER_NAME` name.

### SSO Authentication

The script now supports browser-based SSO authentication via OpenShift's OAuth server. This is useful when your OpenShift cluster is configured with an identity provider like Keycloak, Okta, Azure AD, or others.

#### How SSO Works

1. When authentication is needed, the script will automatically open your default browser to the OpenShift OAuth login page
2. You'll authenticate with your SSO provider (e.g., Keycloak, Okta)
3. After successful authentication, you'll receive either an authorization code or access token
4. Copy the code/token and paste it into the terminal prompt
5. The script will exchange the code for a token (if needed) and cache it for future use

#### SSO Configuration

By default, SSO authentication is **enabled** and will be attempted after checking for a cached token, but before falling back to username/password authentication.

To **disable SSO** authentication:
```bash
export SSO_ENABLED=false
```

To specify a custom OAuth client ID (if your cluster uses a different client):
```bash
export SSO_CLIENT_ID=my-custom-client
```

#### Example kubeconfig with SSO

```yaml
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: /path/to/openshift-auth-plugin.sh
      env:
        - name: "CLUSTER_NAME"
          value: "my-openshift-cluster"
        - name: "SSO_ENABLED"
          value: "true"
      provideClusterInfo: true
```

#### SSO Requirements

- **OAuth Client**: Your OpenShift cluster must have an OAuth client configured (typically `openshift-browser-client`)
- **Browser**: A web browser must be available on your system
- **Interactive Terminal**: The script needs to prompt for user input (not suitable for CI/CD without `SSO_ENABLED=false`)

#### Troubleshooting SSO

**Browser doesn't open automatically:**
- The script will print the authorization URL - copy and paste it into your browser manually

**"Invalid authorization code" error:**
- Make sure you're copying the entire authorization code without extra spaces
- The code is typically valid for only a few minutes - don't wait too long

**SSO not working in CI/CD:**
- Set `SSO_ENABLED=false` in non-interactive environments
- Use username/password authentication or pre-configured tokens instead

**Token expires too quickly:**
- This is controlled by your OpenShift OAuth configuration and identity provider settings
- The script will automatically re-authenticate when the cached token expires
