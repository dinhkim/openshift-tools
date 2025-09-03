# OpenShift Tools

This repository contains a collection of tools for working with OpenShift.

## openshift-auth-plugin.sh

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

The script can be configured to authenticate in two ways:

1.  **Token Authentication**: If a valid, unexpired token is found in the configured secret store (Keychain or `gopass`), it will be used for authentication.
2.  **Username/Password Authentication**: If a token is not available or has expired, the script will try to authenticate using a username and password from the secret store.

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
# Store your username
security add-generic-password -a $USER -s "kubeconfig-my-openshift-cluster-username" -w "my-username"

# Store your password
security add-generic-password -a $USER -s "kubeconfig-my-openshift-cluster-password" -w "my-password"
```

#### gopass

```bash
# Store your username
gopass insert -f my-openshift-cluster/username

# Store your password
gopass insert -f my-openshift-cluster/password
```

Replace `my-openshift-cluster` with your `CLUSTER_NAME` name.
