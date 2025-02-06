#!/bin/bash

# Print installation instructions for Docker
print_docker_instructions() {
    echo "Docker installation instructions:"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "[INFO] macOS: Install Docker Desktop"
        echo "1. Visit: https://www.docker.com/products/docker-desktop"
        echo "2. Download and install Docker Desktop for Mac"
        echo "3. Start Docker Desktop after installation"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "[INFO] Linux: Install Docker Engine"
        echo "1. Visit: https://docs.docker.com/engine/install/"
        echo "2. Follow installation instructions for your distribution"
        echo "3. Run these commands after installation:"
        echo "   sudo systemctl enable docker"
        echo "   sudo systemctl start docker"
        echo "   sudo usermod -aG docker \$USER"
        echo "4. Log out and log back in for group changes to take effect"
    else
        echo "[ERROR] Unsupported operating system"
        exit 1
    fi
}

# Print installation instructions for Docker Compose
print_compose_instructions() {
    echo "Docker Compose installation instructions:"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "[INFO] Docker Compose comes with Docker Desktop for Mac"
        echo "Install Docker Desktop to get Docker Compose"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "[INFO] Linux: Install Docker Compose plugin"
        echo "1. Visit: https://docs.docker.com/compose/install/linux/"
        echo "2. Follow the instructions to install Docker Compose plugin"
    else
        echo "[ERROR] Unsupported operating system"
        exit 1
    fi
}

# Get and export version information
export_versions() {
    echo "[INFO] Reading project versions..."
    
    SCRIPT_PATH=$(cd "$(dirname "$0")" && pwd)
    PROJECT_ROOT=$(cd "${SCRIPT_PATH}/.." && pwd)
    
    # Get Go version from go.mod
    if [ -f "${PROJECT_ROOT}/go.mod" ]; then
        GO_VERSION=$(grep -E "^go [0-9]+\.[0-9]+\.[0-9]+" "${PROJECT_ROOT}/go.mod" | cut -d" " -f2)
        echo "[OK] Go version: ${GO_VERSION}"
        echo "GO_VERSION=${GO_VERSION}" > "${SCRIPT_PATH}/.env"
    else
        echo "[ERROR] go.mod not found at ${PROJECT_ROOT}/go.mod"
        exit 1
    fi
    
    # Get Node version from .node-version
    if [ -f "${PROJECT_ROOT}/.node-version" ]; then
        FULL_NODE_VERSION=$(cat "${PROJECT_ROOT}/.node-version")
        NODE_VERSION=$(echo "$FULL_NODE_VERSION" | cut -d. -f1)
        echo "[OK] Node version: ${NODE_VERSION}"
        echo "NODE_VERSION=${NODE_VERSION}" >> "${SCRIPT_PATH}/.env"
    else
        echo "[ERROR] .node-version not found at ${PROJECT_ROOT}/.node-version"
        exit 1
    fi
    
    # Also export for current session
    export GO_VERSION NODE_VERSION
}

check_prerequisites() {
    local missing_tools=()
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        missing_tools+=("Docker")
    else
        echo "[OK] Docker is installed"
        # Check Docker service using socket
        if ! docker version &> /dev/null || ! test -S /var/run/docker.sock; then
            echo "[ERROR] Docker service is not running"
            missing_tools+=("Docker service")
        fi
    fi
    
    # Check Docker Compose
    if ! (command -v docker-compose &> /dev/null || docker compose version &> /dev/null); then
        missing_tools+=("Docker Compose")
    else
        echo "[OK] Docker Compose is installed"
    fi
    
    # Handle missing tools
    if [ ${#missing_tools[@]} -ne 0 ]; then
        echo "[ERROR] Missing required tools:"
        for tool in "${missing_tools[@]}"; do
            echo "  - $tool"
            if [ "$tool" == "Docker" ]; then
                print_docker_instructions
            elif [ "$tool" == "Docker Compose" ]; then
                print_compose_instructions
            fi
        done
        exit 1
    fi
    
    # After all prerequisites are met, export versions
    export_versions
    
    echo "[OK] All prerequisites are met!"
}

echo "[INFO] Checking development environment prerequisites..."
check_prerequisites