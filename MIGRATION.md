# Migration Guide: Shell Script → Go Plugin

This guide helps you migrate from the shell script (`openshift-auth-plugin.sh`) to the Go plugin (`openshift-auth-plugin`).

## Why Migrate?

| Aspect | Shell Script | Go Plugin |
|--------|-------------|-----------|
| **Performance** | ~500ms | ~50ms (10x faster) |
| **Dependencies** | curl, jq, base64, date | None |
| **Error Handling** | Basic | Comprehensive with context |
| **Cross-platform** | macOS/Linux only | macOS/Linux/Windows |
| **Maintenance** | Harder to debug | Easier with type safety |
| **Binary Size** | N/A | ~15MB (statically linked) |

## Prerequisites

1. **Go 1.21+** installed (for building)
2. **Existing shell script** working configuration
3. **Backup** of your `~/.kube/config`

## Migration Steps

### Step 1: Build the Go Plugin

```bash
cd /path/to/openshift-tools

# Download dependencies
go mod download

# Build
make build

# Install
sudo make install
```

Verify installation:
```bash
which openshift-auth-plugin
# Should output: /usr/local/bin/openshift-auth-plugin
```

### Step 2: Update Kubeconfig

**Before (Shell Script):**
```yaml
users:
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

**After (Go Plugin):**
```yaml
users:
- name: my-openshift-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1  # Note: v1, not v1beta1
      command: openshift-auth-plugin              # No path needed
      env:
        - name: CLUSTER_NAME                      # No quotes needed
          value: my-openshift-cluster
        - name: DEBUG
          value: "false"
      provideClusterInfo: true
      interactiveMode: IfAvailable                # Optional but recommended
```

**Key Changes:**
1. `apiVersion`: Changed from `v1beta1` to `v1`
2. `command`: Changed from full path to just `openshift-auth-plugin`
3. `env`: Removed quotes around environment variable names
4. Added `interactiveMode: IfAvailable` (optional)

### Step 3: Migrate Stored Credentials

The Go plugin uses the **same storage format** as the shell script, so your existing tokens will work!

**No action needed** - your cached tokens in Keychain or gopass will be automatically used.

To verify:
```bash
# Check stored token (macOS Keychain)
security find-generic-password -a "openshift-auth-plugin" -s "my-cluster-token" -w

# Check stored token (gopass)
gopass show my-cluster/token
```

### Step 4: Test the Migration

```bash
# Enable debug mode for first test
export DEBUG=true

# Test with kubectl
kubectl get pods

# You should see debug output from the Go plugin
```

Expected output:
```
2026-02-14T12:00:00Z Starting OpenShift authentication plugin
2026-02-14T12:00:00Z Cluster: my-cluster, Server: https://api.cluster.example.com:6443
2026-02-14T12:00:00Z Retrieving token from keychain
2026-02-14T12:00:00Z Validating token with: https://api.cluster.example.com:6443/apis/user.openshift.io/v1/users/~
2026-02-14T12:00:00Z Token validation successful
2026-02-14T12:00:00Z Using cached token
```

### Step 5: Disable Debug Mode

Once verified, disable debug mode:

```yaml
env:
- name: DEBUG
  value: "false"  # or remove this line entirely
```

## Configuration Mapping

All environment variables from the shell script are supported:

| Shell Script | Go Plugin | Notes |
|-------------|-----------|-------|
| `OPENSHIFT_URL` | `OPENSHIFT_URL` | Same |
| `CLUSTER_NAME` | `CLUSTER_NAME` | Same |
| `VERIFY_SSL` | `VERIFY_SSL` | Same |
| `SECRET_STORE` | `SECRET_STORE` | Same (`keychain` or `gopass`) |
| `DEBUG` | `DEBUG` | Same |
| `SSO_ENABLED` | `SSO_ENABLED` | Same (default: `true`) |
| `SSO_CLIENT_ID` | `SSO_CLIENT_ID` | Same |
| `SSO_PROVIDER` | `SSO_PROVIDER` | Same (display only) |

## Rollback Plan

If you need to rollback to the shell script:

### Step 1: Revert Kubeconfig

```yaml
users:
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

### Step 2: Test

```bash
kubectl get pods
```

Your cached tokens will still work with the shell script.

## Side-by-Side Comparison

You can run both implementations side-by-side for different clusters:

```yaml
users:
# Cluster 1: Using Go plugin
- name: cluster1-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: openshift-auth-plugin
      env:
      - name: CLUSTER_NAME
        value: cluster1
      provideClusterInfo: true

# Cluster 2: Using shell script
- name: cluster2-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: /path/to/openshift-auth-plugin.sh
      env:
      - name: CLUSTER_NAME
        value: cluster2
      provideClusterInfo: true
```

## Troubleshooting

### "command not found: openshift-auth-plugin"

**Solution**: Install the binary to your PATH:
```bash
sudo make install
# or
sudo cp openshift-auth-plugin /usr/local/bin/
```

### "failed to load configuration: CLUSTER_NAME is required"

**Solution**: Check your kubeconfig env section has `CLUSTER_NAME` set:
```yaml
env:
- name: CLUSTER_NAME
  value: my-cluster
```

### Token validation fails after migration

**Solution**: Delete cached token and re-authenticate:
```bash
security delete-generic-password -a "openshift-auth-plugin" -s "my-cluster-token"
kubectl get pods
```

### Different behavior between implementations

**Solution**: Enable debug mode on both and compare:
```bash
# Shell script
DEBUG=true /path/to/openshift-auth-plugin.sh

# Go plugin
DEBUG=true openshift-auth-plugin
```

## Performance Comparison

Test the performance difference:

```bash
# Shell script
time /path/to/openshift-auth-plugin.sh
# Typical: 0.5-1.0 seconds

# Go plugin
time openshift-auth-plugin
# Typical: 0.05-0.1 seconds
```

## Feature Parity

Both implementations support:

✅ OAuth/SSO authentication  
✅ Token caching (Keychain/gopass)  
✅ Token validation  
✅ SSL verification control  
✅ Debug logging  
✅ ExecCredential API  
✅ Multiple clusters  

The Go plugin additionally provides:

✅ Better error messages with context  
✅ Type safety  
✅ Faster execution  
✅ No external dependencies  
✅ Easier to test and maintain  

## Post-Migration Cleanup

After successful migration, you can optionally:

1. **Remove shell script** (keep for rollback initially):
   ```bash
   # After 1-2 weeks of successful use
   rm /path/to/openshift-auth-plugin.sh
   ```

2. **Update documentation** for your team

3. **Share the migration guide** with other users

## Getting Help

If you encounter issues during migration:

1. **Check logs**: Enable `DEBUG=true`
2. **Verify configuration**: Compare with examples in this guide
3. **Test manually**: Run the plugin directly with env vars
4. **Rollback if needed**: Use the rollback plan above
5. **Report issues**: Create an issue in the repository

## Migration Checklist

- [ ] Go 1.21+ installed
- [ ] Backup of `~/.kube/config` created
- [ ] Go plugin built and installed
- [ ] Kubeconfig updated (apiVersion, command)
- [ ] Tested with `kubectl get pods`
- [ ] Debug mode disabled
- [ ] Documented changes for team
- [ ] Rollback plan understood
- [ ] Shell script kept for 1-2 weeks (safety)

## Next Steps

After successful migration:

- Read [README-GO.md](README-GO.md) for full documentation
- Check [AGENTS-GO.md](AGENTS-GO.md) for development guidelines
- Consider contributing improvements back to the project
