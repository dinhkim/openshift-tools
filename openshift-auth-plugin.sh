#!/bin/bash

# OpenShift Kubeconfig Exec Plugin
# This script is designed to be used as a kubeconfig exec plugin for OpenShift authentication

set -euo pipefail

# Configuration from environment variables
OPENSHIFT_URL="${KUBERNETES_EXEC_INFO_OPENSHIFT_URL:-${OPENSHIFT_URL:-}}"
OPENSHIFT_USERNAME="${KUBERNETES_EXEC_INFO_OPENSHIFT_USERNAME:-${OPENSHIFT_USERNAME:-}}"
OPENSHIFT_PASSWORD="${KUBERNETES_EXEC_INFO_OPENSHIFT_PASSWORD:-${OPENSHIFT_PASSWORD:-}}"
OPENSHIFT_TOKEN="${KUBERNETES_EXEC_INFO_OPENSHIFT_TOKEN:-${OPENSHIFT_TOKEN:-}}"
CONTEXT="${KUBERNETES_EXEC_INFO_CONTEXT:-${CONTEXT:-}}"
VERIFY_SSL="${KUBERNETES_EXEC_INFO_VERIFY_SSL:-${VERIFY_SSL:-false}}"

# Set curl options
CURL_OPTS=""
if [[ "$VERIFY_SSL" == "false" ]]; then
    CURL_OPTS="-k"
fi

# Function to output error and exit
error_exit() {
    local message="$1"
    cat << EOF >&2
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "status": {
    "error": "$message"
  }
}
EOF
    exit 1
}

# Function to output successful token
output_token() {
    local token="$1"
    local expiry="$2"
    
    cat << EOF
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "spec": {
    "interactive": false
  },
  "status": {
    "token": "$token",
    "expirationTimestamp": "$expiry"
  }
}
EOF
}

# Function to get OAuth info
get_oauth_info() {
    local server_url="$1"
    local oauth_url="${server_url}/.well-known/oauth-authorization-server"
    
    local response
    if ! response=$(curl -s $CURL_OPTS "$oauth_url" 2>/dev/null); then
        error_exit "Failed to connect to OAuth endpoint"
    fi
    
    if ! echo "$response" | jq . >/dev/null 2>&1; then
        error_exit "Invalid JSON response from OAuth endpoint"
    fi
    
    echo "$response"
}

# Function to authenticate with credentials
authenticate_with_credentials() {
    local server_url="$1"
    local username="$2"
    local password="$3"
    
    # Get OAuth information
    local oauth_info
    if ! oauth_info=$(get_oauth_info "$server_url"); then
        return 1
    fi
    
    # Extract authorization endpoint
    local authorization_endpoint
    if ! authorization_endpoint=$(echo "$oauth_info" | jq -r '.authorization_endpoint'); then
        error_exit "Could not parse authorization endpoint"
    fi
    
    if [[ "$authorization_endpoint" == "null" ]]; then
        error_exit "Authorization endpoint not found"
    fi
    
    # Encode credentials
    local credentials
    credentials=$(echo -n "${username}:${password}" | base64 -w 0)
    
    # Request token
    local response
    response=$(curl -v -s $CURL_OPTS -I \
        -X GET \
        -H "Authorization: Basic ${credentials}" \
        -H "X-CSRF-Token: XXXXX" \
        "$authorization_endpoint?client_id=openshift-challenging-client&response_type=token" 2>/dev/null)

    
    if [[ $? -ne 0 ]]; then
        error_exit "Failed to connect to authorization endpoint"
    fi
    
    # Extract access token
    local access_token
    local location_header
    location_header=$(echo "$response" | grep -i "^Location:" | head -1)
    
    if [[ -n "$location_header" ]]; then
        # Extract access_token from URL fragment
        access_token=$(echo "$location_header" | sed -n 's/.*access_token=\([^&]*\).*/\1/p' 2>/dev/null || echo "")
    fi
    
    if [[ "$access_token" == "null" ]] || [[ -z "$access_token" ]]; then
        local error_msg
        error_msg=$(echo "$response" | grep -i "^Response:" | head -1 2>/dev/null)
        error_exit "Authentication failed: $error_msg"
    fi
    
    # Extract expiry (if provided)
    local expires_in
    expires_in=$(echo "$location_header" | sed -n 's/.*expires_in=\([^&]*\).*/\1/p' 2>/dev/null || echo "")
    
    local expiry_timestamp=""
    if [[ -n "$expires_in" && "$expires_in" != "null" ]]; then
        # Calculate expiry timestamp (current time + expires_in seconds)
        expiry_timestamp=$(date -v"+${expires_in}S" -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "")
    fi
    
    # Default expiry to 24 hours if not provided
    if [[ -z "$expiry_timestamp" ]]; then
        expiry_timestamp=$(date -v"+24H" -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+1 day" +"%Y-%m-%dT%H:%M:%SZ")
    fi
    
    # Store token in gopass
    echo -n "$access_token" | gopass insert -f "$CONTEXT" token || return 1
    echo -n "$expiry_timestamp" | gopass insert -f "$CONTEXT" token_expiry || return 1

    output_token "$access_token" "$expiry_timestamp"
}

# Function to validate token
validate_token() {
    local server_url="$1"
    local token="$2"
    
    # Test the token
    local user_url="${server_url}/apis/user.openshift.io/v1/users/~"
    local response
    local http_code
    
    response=$(curl -s $CURL_OPTS \
        -w "%{http_code}" \
        -H "Authorization: Bearer ${token}" \
        -H "Accept: application/json" \
        "$user_url" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        error_exit "Failed to validate token"
    fi
    
    # Extract HTTP status code
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        return 0
    else
        # error_exit "Token validation failed (HTTP $http_code)"
        return 1
    fi
}

# Function to check dependencies
check_dependencies() {
    local deps=("curl" "jq" "base64" "date" "gopass")
    local missing_deps=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error_exit "Missing required dependencies: ${missing_deps[*]}"
    fi
}

# Main execution
main() {
    # Check dependencies
    check_dependencies
    
    # Validate required configuration
    if [[ -z "$OPENSHIFT_URL" ]]; then
        error_exit "OPENSHIFT_URL environment variable is required"
    fi

    # Validate required context
    if [[ -z "$CONTEXT" ]]; then
        error_exit "CONTEXT environment variable is required"
    fi

    # Method 1: Use existing token
    # Get token stored in gopass
    if [[ -z "$OPENSHIFT_TOKEN" ]]; then
        OPENSHIFT_TOKEN=$(gopass show -o "$CONTEXT" token 2>/dev/null || return 0)
    fi

    if [[ -n "$OPENSHIFT_TOKEN" ]]; then
        if $(validate_token "$OPENSHIFT_URL" "$OPENSHIFT_TOKEN"); then
            local expiry_timestamp=$(gopass show -o "$CONTEXT" token_expiry 2>/dev/null || return 0)
            output_token "$OPENSHIFT_TOKEN" "$expiry_timestamp"
            return
        fi
    fi

    # Method 2: Use username/password
    # Get username and password stored in gopass
    if [[ -z "$OPENSHIFT_USERNAME" ]]; then
        OPENSHIFT_USERNAME=$(gopass show -o "$CONTEXT" username 2>/dev/null || return 0)
    fi

    if [[ -z "$OPENSHIFT_PASSWORD" ]]; then
        OPENSHIFT_PASSWORD=$(gopass show -o "$CONTEXT" password 2>/dev/null || return 0)
    fi

    if [[ -n "$OPENSHIFT_USERNAME" && -n "$OPENSHIFT_PASSWORD" ]]; then
        authenticate_with_credentials "$OPENSHIFT_URL" "$OPENSHIFT_USERNAME" "$OPENSHIFT_PASSWORD"
        return
    fi
    
    # No valid authentication method
    error_exit "No valid authentication method found. Set OPENSHIFT_TOKEN or OPENSHIFT_USERNAME/OPENSHIFT_PASSWORD"
}

# Execute main function
main "$@"