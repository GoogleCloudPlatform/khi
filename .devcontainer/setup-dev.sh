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
    go install github.com/google/addlicense@latest
    log "Go setup complete"
}

# Setup Web environment
setup_web() {
    log "Setting up Web environment..."

    # Install dev tools
    npm install --cache /tmp/cache

    cd web 
    # Install project dependencies
    for i in {1..3}; do
        if npm install --cache /tmp/cache; then
            rm -rf /tmp/cache
            break
        fi
        log "Retry $i: npm install failed, retrying..."
        sleep 5
    done

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

main() {
    setup_go || exit 1
    setup_web || exit 1
    setup_git || exit 1
    log "Development environment setup complete!"
}

main "$@"
