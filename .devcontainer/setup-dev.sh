#!/bin/bash
set -e

log() {
    printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$1"
}

# Setup Go environment
setup_go() {
    log "Setting up Go environment..."
    if ! go mod download; then
        log "Failed to download Go modules"
        return 1
    fi
    go mod verify
    go install golang.org/x/tools/cmd/goimports@latest
    log "Go setup complete"
}

# Setup Web environment
setup_web() {
    log "Setting up Web environment..."
    cd web || { log "Web directory not found"; return 1; }

    # Install project dependencies
    for i in {1..3}; do
        if npm install --cache /tmp/cache; then
            rm -rf /tmp/cache
            break
        fi
        log "Retry $i: npm install failed, retrying..."
        sleep 5
    done

    # Install Angular CLI only if not already installed
    if ! command -v ng &>/dev/null; then
        npm install -g --prefix ~/.npm-global @angular/cli || {
            log "Failed to install Angular CLI globally"
        }
    fi

    cd ..
    log "Web setup complete"
}

# Setup Git configuration
setup_git() {
    log "Configuring Git..."
    git config --global core.editor 'vim'
    git config --global commit.gpgsign true
    git config --global pull.rebase true
    git config --global core.autocrlf input
    log "Git setup complete"
}

# Verify required tools
verify_tools() {
    log "Verifying development tools..."
    local missing_tools=()
    
    if ! gcloud --version &>/dev/null; then
        missing_tools+=("gcloud")
    fi
    
    if ! kubectl version --client &>/dev/null; then
        missing_tools+=("kubectl")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log "Missing required tools:"
        printf '%s\n' "${missing_tools[@]}"
        return 1
    fi
    
    log "All tools verified"
}

main() {
    verify_tools || exit 1
    setup_go || exit 1
    setup_web || exit 1
    setup_git || exit 1
    log "Development environment setup complete!"
}

main "$@"