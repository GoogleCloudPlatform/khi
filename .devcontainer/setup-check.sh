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

# Installation instructions
print_instructions() {
    local tool=$1
    case "$tool" in
        "Docker")
            log "INFO" "To install Docker:"
            echo "1. Visit Docker's official website: https://www.docker.com/get-started"
            echo "2. Download Docker Desktop (for Mac/Windows) or Docker Engine (for Linux)"
            echo "3. Follow the installation guide for your operating system"
            echo "4. Start Docker and verify installation with: docker --version"
            ;;
        "Docker Compose")
            log "INFO" "To install Docker Compose:"
            echo "1. Docker Compose comes with Docker Desktop"
            echo "2. For Linux, visit: https://docs.docker.com/compose/install/"
            echo "3. Verify installation with: docker compose version"
            ;;
    esac
}

# Version and hash management
verify_hash() {
    local version=$1
    local type=$2
    local hash_url=""
    local hash=""
    
    case "$type" in
        "go")
            hash_url="https://dl.google.com/go/go${version}.linux-amd64.tar.gz.sha256"
            log "INFO" "Fetching Go hash from: ${hash_url}" >&2
            hash=$(safe_curl "${hash_url}" 2>/dev/null | tr -d '%' | grep -o '[a-f0-9]\{64\}')
            ;;
        "node")
            hash_url="https://nodejs.org/dist/v${version}/SHASUMS256.txt"
            log "INFO" "Fetching Node.js hashes from: ${hash_url}" >&2
            local response=$(safe_curl "${hash_url}" 2>/dev/null)
            amd64_hash=$(echo "$response" | grep "node-v${version}-linux-x64.tar.xz" | awk '{print $1}')
            arm64_hash=$(echo "$response" | grep "node-v${version}-linux-arm64.tar.xz" | awk '{print $1}')
            hash="${amd64_hash}:${arm64_hash}"
            ;;
        "kubectl")
            hash_url="https://dl.k8s.io/release/v${version}/bin/linux/amd64/kubectl.sha256"
            log "INFO" "Fetching kubectl hash from: ${hash_url}" >&2
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
    local mode=${5:-">>"}
    
    if [ "$type" = "NODE" ]; then
        # Handle Node.js special case with both hashes
        local amd64_hash=$(echo "${sha256}" | cut -d':' -f1)
        local arm64_hash=$(echo "${sha256}" | cut -d':' -f2)
        if [ "$mode" = ">" ]; then
            {
                echo "${type}_VERSION=${version}"
                echo "${type}_AMD64_SHA256=${amd64_hash}"
                echo "${type}_ARM64_SHA256=${arm64_hash}"
            } > "${env_file}"
        else
            {
                echo "${type}_VERSION=${version}"
                echo "${type}_AMD64_SHA256=${amd64_hash}"
                echo "${type}_ARM64_SHA256=${arm64_hash}"
            } >> "${env_file}"
        fi
    else
        # Standard handling for other tools
        if [ "$mode" = ">" ]; then
            {
                echo "${type}_VERSION=${version}"
                echo "${type}_SHA256=${sha256}"
            } > "${env_file}"
        else
            {
                echo "${type}_VERSION=${version}"
                echo "${type}_SHA256=${sha256}"
            } >> "${env_file}"
        fi
    fi
}

export_versions() {
    log "INFO" "Reading project versions..."
    
    SCRIPT_PATH=$(cd "$(dirname "$0")" && pwd)
    PROJECT_ROOT=$(cd "${SCRIPT_PATH}/.." && pwd)
    ENV_FILE="${SCRIPT_PATH}/.env"

    # Initialize .env file with user information
    {
        # User configuration
        echo "USERNAME=$(id -un)"
        echo "USER_UID=$(id -u)"
        echo "USER_GID=$(id -g)"
    } > "${ENV_FILE}"
    
    # Get Go version and hash
    if [ -f "${PROJECT_ROOT}/go.mod" ]; then
        GO_VERSION=$(grep -E "^go [0-9]+\.[0-9]+\.[0-9]+" "${PROJECT_ROOT}/go.mod" | cut -d" " -f2)
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
        # Split the combined hash for export
        NODE_AMD64_SHA256=$(echo "${NODE_SHA256}" | cut -d':' -f1)
        NODE_ARM64_SHA256=$(echo "${NODE_SHA256}" | cut -d':' -f2)
    else
        log "ERROR" ".node-version not found at ${PROJECT_ROOT}/.node-version"
        exit 1
    fi
    
    # Get kubectl version and hash
    KUBECTL_VERSION=$(safe_curl "https://dl.k8s.io/release/stable.txt" | sed 's/v//')
    KUBECTL_SHA256=$(verify_hash "${KUBECTL_VERSION}" "kubectl")
    log "OK" "kubectl version: ${KUBECTL_VERSION}"
    write_version_to_env "KUBECTL" "${KUBECTL_VERSION}" "${KUBECTL_SHA256}" "${ENV_FILE}"
    
    # Collect all variables to export
    export_vars=(
        GO_VERSION GO_SHA256
        NODE_VERSION NODE_AMD64_SHA256 NODE_ARM64_SHA256
        KUBECTL_VERSION KUBECTL_SHA256
    )
    
    export "${export_vars[@]}"
}

check_prerequisites() {
    local missing_tools=()
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        missing_tools+=("Docker")
    else
        log "OK" "Docker is installed"
        if ! docker version &> /dev/null || ! test -S /var/run/docker.sock; then
            log "ERROR" "Docker service is not running"
            missing_tools+=("Docker service")
        fi
    fi
    
    # Check Docker Compose
    if ! (docker compose version &> /dev/null); then
        missing_tools+=("Docker Compose")
    else
        log "OK" "Docker Compose is installed"
    fi
    
    # Handle missing tools
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log "ERROR" "Missing required tools:"
        for tool in "${missing_tools[@]}"; do
            echo "  - $tool"
            print_instructions "$tool"
        done
        exit 1
    fi
    
    # After all prerequisites are met, export versions
    export_versions
    
    log "OK" "All prerequisites are met!"
}

log "INFO" "Checking development environment prerequisites..."
check_prerequisites