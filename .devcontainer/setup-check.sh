#!/bin/bash
# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Set error handling
set -euo pipefail

# Utility functions
log() {
    local level=$1
    local message=$2
    echo "[${level}] ${message}"
}

safe_curl() {
    local url=$1
    curl -sSL --fail --retry 3 --retry-delay 2 --connect-timeout 10 --max-time 15 "${url}"
}

# Version and hash management
verify_hash() {
    local version=$1
    local type=$2
    local host_arch=$(uname -m)
    local hash_url=""
    local hash=""
    local node_arch=""
    local k8s_arch=""

    # Determine architecture strings for URLs/filenames
    case "$host_arch" in
        x86_64)
            node_arch="linux-x64"
            k8s_arch="amd64"
            ;;
        aarch64 | arm64)
            node_arch="linux-arm64"
            k8s_arch="arm64"
            ;;
        *)
            log "ERROR" "Unsupported host architecture for hash verification: ${host_arch}" >&2
            exit 1
            ;;
    esac

    case "$type" in
        "go")
            # Go uses 'amd64' and 'arm64' in the URL path, matching k8s_arch
            hash_url="https://dl.google.com/go/go${version}.linux-${k8s_arch}.tar.gz.sha256"
            log "INFO" "Fetching Go hash (${k8s_arch}) from: ${hash_url}" >&2
            hash=$(safe_curl "${hash_url}" 2>/dev/null | tr -d '%' | grep -o '[a-f0-9]\{64\}')
            ;;
        "node")
            hash_url="https://nodejs.org/dist/v${version}/SHASUMS256.txt"
            log "INFO" "Fetching Node.js hashes from: ${hash_url}" >&2
            local response=$(safe_curl "${hash_url}" 2>/dev/null)
            # Extract the specific hash based on node_arch
            hash=$(echo "$response" | grep "node-v${version}-${node_arch}.tar.xz" | awk '{print $1}')
            log "INFO" "Selected Node.js hash for ${node_arch}" >&2
            ;;
        "kubectl")
            hash_url="https://dl.k8s.io/release/v${version}/bin/linux/${k8s_arch}/kubectl.sha256"
            log "INFO" "Fetching kubectl hash (${k8s_arch}) from: ${hash_url}" >&2
            hash=$(safe_curl "${hash_url}" 2>/dev/null | tr -d ' \t\n\r')
            ;;
    esac

    if [ -z "$hash" ]; then
        log "ERROR" "Failed to fetch hash for ${type} version ${version}" >&2
        exit 1
    fi
    
    echo "$hash"
}

write_version_to_env() {
    local type=$1
    local version=$2
    local sha256=$3
    local env_file=$4
    {
        echo "${type}_VERSION=${version}"
        echo "${type}_SHA256=${sha256}"
    } >> "${env_file}"
}

export_versions() {
    log "INFO" "Reading project versions..."
    
    SCRIPT_PATH=$(cd "$(dirname "$0")" && pwd)
    PROJECT_ROOT=$(cd "${SCRIPT_PATH}/.." && pwd)
    ENV_FILE="${SCRIPT_PATH}/.env"

    # Initialize .env file with user information
    USERNAME=$(id -un)
    if [ "${USERNAME}" = "root" ]; then \
        echo "USERNAME is 'root'. Creating 'developer' user instead." >&2; \
        USERNAME="developer"; \
    fi
    echo "USERNAME=$USERNAME" > "${ENV_FILE}"

    # Get Go version and hash
    if [ -f "${PROJECT_ROOT}/go.mod" ]; then
        GO_VERSION=$(grep -E "^toolchain go[0-9]+\.[0-9]+\.[0-9]+" "${PROJECT_ROOT}/go.mod" | sed -E 's/^toolchain go([0-9]+\.[0-9]+\.[0-9]+)$/\1/')
        GO_SHA256=$(verify_hash "${GO_VERSION}" "go")
        log "OK" "Go version: ${GO_VERSION}"
        write_version_to_env "GO" "${GO_VERSION}" "${GO_SHA256}" "${ENV_FILE}"
    else
        log "ERROR" "go.mod not found at ${PROJECT_ROOT}/go.mod"
        exit 1
    fi
    
    # Get Node version and hash
    if [ -f "${PROJECT_ROOT}/.node-version" ]; then
        NODE_VERSION=$(cat "${PROJECT_ROOT}/.node-version")
        NODE_SHA256=$(verify_hash "${NODE_VERSION}" "node")
        log "OK" "Node version: ${NODE_VERSION}"
        write_version_to_env "NODE" "${NODE_VERSION}" "${NODE_SHA256}" "${ENV_FILE}"
    else
        log "ERROR" ".node-version not found at ${PROJECT_ROOT}/.node-version"
        exit 1
    fi
    
    # Get kubectl version and hash
    KUBECTL_VERSION=$(safe_curl "https://dl.k8s.io/release/stable.txt" | sed 's/v//')
    KUBECTL_SHA256=$(verify_hash "${KUBECTL_VERSION}" "kubectl")
    log "OK" "kubectl version: ${KUBECTL_VERSION}"
    write_version_to_env "KUBECTL" "${KUBECTL_VERSION}" "${KUBECTL_SHA256}" "${ENV_FILE}"
}

log "INFO" "Checking development environment prerequisites..."
export_versions
