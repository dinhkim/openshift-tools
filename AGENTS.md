# Agent Guidelines for OpenShift Tools

This document provides coding guidelines and context for AI coding agents working in this repository.

## Project Overview

**Type**: Shell script utility  
**Purpose**: OpenShift authentication plugin for kubeconfig that handles token-based and credential-based authentication  
**Main file**: `openshift-auth-plugin.sh` (309 lines)  
**Language**: Bash/Shell  
**Dependencies**: curl, jq, base64, date, gopass (optional), security (macOS Keychain)

## Build, Test, and Lint Commands

### Testing
There are currently **no automated tests** in this project. When testing changes:

```bash
# Manual testing - ensure script is executable
chmod +x openshift-auth-plugin.sh

# Syntax check
bash -n openshift-auth-plugin.sh

# Test script with debug output
DEBUG=true CLUSTER_NAME=test-cluster ./openshift-auth-plugin.sh

# Validate JSON output format (should match ExecCredential spec)
DEBUG=false CLUSTER_NAME=test-cluster ./openshift-auth-plugin.sh | jq .
```

### Linting
No formal linter is configured. Use shellcheck if available:

```bash
# Install shellcheck (macOS)
brew install shellcheck

# Run shellcheck
shellcheck openshift-auth-plugin.sh
```

### Running Single Tests
N/A - no test framework exists. Manual testing only.

## Code Style Guidelines

### Shell Script Standards

1. **Shebang and Safety**
   - Always use `#!/bin/bash` as the first line
   - Include `set -euo pipefail` early for error handling
   - `set -e`: Exit on error
   - `set -u`: Treat unset variables as errors
   - `set -o pipefail`: Fail on pipe errors

2. **Variable Conventions**
   - Environment variables: `UPPER_SNAKE_CASE` (e.g., `OPENSHIFT_URL`, `SECRET_STORE`)
   - Local variables: `lower_snake_case` (e.g., `server_url`, `access_token`)
   - Always use `local` for function-scoped variables
   - Quote all variable expansions: `"$variable"` not `$variable`
   - Provide defaults for env vars: `${VARIABLE:-default_value}`

3. **Function Naming and Structure**
   - Function names: `lower_snake_case` (e.g., `get_oauth_info`, `validate_token`)
   - Private/helper functions: prefix with `_` (e.g., `_get_secret`, `_store_secret`)
   - Document complex functions with comments
   - Use `local` for all function parameters and variables

4. **Function Declaration Style**
   ```bash
   # Correct format
   function_name() {
       local param1="$1"
       local param2="$2"
       # function body
   }
   ```

5. **Error Handling**
   - Use `error_exit()` function for fatal errors
   - Output errors to stderr: `>&2`
   - For ExecCredential plugin, errors must follow Kubernetes format (JSON to stderr)
   - Check command success: `if ! command; then ... fi` or `command || error_exit "msg"`
   - Validate JSON with `jq`: `if ! echo "$response" | jq . >/dev/null 2>&1; then`

6. **String Formatting**
   - Use heredocs for multi-line output (JSON, error messages):
     ```bash
     cat << EOF
     {
       "key": "value"
     }
     EOF
     ```
   - Use `echo -n` for no trailing newline
   - Prefer `[[ ]]` over `[ ]` for conditionals

7. **Conditionals and Comparisons**
   - String comparison: `[[ "$var" == "value" ]]`
   - Numeric comparison: `[[ $num -eq 0 ]]`, `[[ $num -gt 5 ]]`
   - Null checks: `[[ -z "$var" ]]` (empty), `[[ -n "$var" ]]` (not empty)
   - File checks: `[[ -f "file" ]]`, `[[ -d "dir" ]]`

8. **Command Execution**
   - Capture output: `variable=$(command)` or `variable=$(command 2>/dev/null)`
   - Silent execution: `command >/dev/null 2>&1`
   - Check if command exists: `command -v "$cmd" >/dev/null 2>&1`
   - Pipe safety: validate each stage, especially with `jq`

9. **Arrays**
   - Declare: `local array=()`
   - Append: `array+=("item")`
   - Iterate: `for item in "${array[@]}"; do ... done`
   - Length: `${#array[@]}`

10. **Logging and Debugging**
    - Use `log()` function for debug output (controlled by `DEBUG` env var)
    - Log format: `$(date -u +'%Y-%m-%dT%H:%M:%SZ') $message`
    - Always log to stderr: `echo "..." >&2`

11. **JSON Handling**
    - Use `jq` for all JSON parsing and validation
    - Extract values: `jq -r '.field'` (`-r` for raw output, no quotes)
    - Validate JSON: `jq . >/dev/null 2>&1`
    - Check for null: `[[ "$value" == "null" ]]`

12. **HTTP Requests**
    - Use `curl` with error handling
    - Standard options: `-s` (silent), `$CURL_OPTS` (SSL verification)
    - Capture HTTP codes: `curl -w "%{{http_code}}"` then extract with `${response: -3}`
    - Always redirect curl errors: `2>/dev/null`

13. **Date/Time Handling**
    - Use ISO 8601 format: `%Y-%m-%dT%H:%M:%SZ`
    - UTC only: `date -u`
    - Parsing dates (macOS): `date -j -f "%Y-%m-%dT%H:%M:%SZ" "$timestamp" +%s`
    - Date arithmetic (macOS): `date -v"+24H"`, `date -v"+${seconds}S"`

14. **Comments**
    - Use `#` for single-line comments
    - Document function purpose, parameters, and return values
    - Explain non-obvious logic
    - Keep comments concise and relevant

15. **Security Considerations**
    - Never log credentials or tokens (unless in DEBUG mode, be careful)
    - Use secure storage (Keychain or gopass) for sensitive data
    - Support SSL verification via `VERIFY_SSL` environment variable
    - Encode credentials properly: `base64 -w 0` for HTTP Basic Auth

## Project-Specific Patterns

### ExecCredential Plugin Format
This script implements the Kubernetes ExecCredential API (v1beta1). Output must be valid JSON:

**Success format:**
```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "status": {
    "token": "...",
    "expirationTimestamp": "2026-02-14T12:00:00Z"
  }
}
```

**Error format (to stderr):**
```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "status": {
    "error": "error message"
  }
}
```

### Authentication Flow
1. Check for cached token → validate → use if valid
2. Fall back to username/password → authenticate → cache token → output
3. Fail with error if no valid method

### Secret Storage Abstraction
Use `_get_secret()` and `_store_secret()` functions. Support both:
- **macOS Keychain**: `security` command
- **gopass**: password manager

## File Structure

```
/
├── README.md                    # User documentation
├── openshift-auth-plugin.sh     # Main executable script
└── AGENTS.md                    # This file (agent guidelines)
```

## Environment Variables

Required in kubeconfig exec config:
- `CLUSTER_NAME`: Cluster identifier for credential storage
- `OPENSHIFT_URL`: OpenShift API URL (or use `provideClusterInfo: true`)

Optional:
- `VERIFY_SSL`: SSL verification (default: `false`)
- `SECRET_STORE`: `keychain` or `gopass` (default: `keychain`)
- `DEBUG`: Enable debug logging (default: `false`)

## Common Tasks

### Adding New Features
1. Follow existing function naming conventions
2. Add error handling with `error_exit()` for fatal errors
3. Use `log()` for debug output
4. Test manually with various configurations
5. Update README.md with new functionality

### Modifying Authentication
- Preserve ExecCredential JSON format
- Validate tokens before output
- Store credentials securely
- Handle errors gracefully (return JSON error, not bash errors)

### Dependencies
Before adding new dependencies:
1. Check if already available in macOS/Linux by default
2. Add to `check_dependencies()` function
3. Document in README.md prerequisites

## Best Practices

- **Always test changes manually** - no CI/CD exists
- **Preserve backward compatibility** - users rely on existing behavior
- **Keep it portable** - support both macOS and Linux date commands
- **Security first** - never expose credentials in logs or output
- **Fail safely** - return proper ExecCredential error format
- **Document changes** - update README.md for user-facing changes
