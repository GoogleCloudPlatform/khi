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

ARG UBUNTU_VERSION
FROM ubuntu:${UBUNTU_VERSION} as base

# Arguments
ARG GO_VERSION
ARG NODE_VERSION
ARG KUBECTL_VERSION
ARG GO_SHA256
ARG NODE_SHA256
ARG KUBECTL_SHA256

# Install system packages
RUN apt-get update && apt-get install -y \
    apt-transport-https \
    build-essential \
    ca-certificates \
    curl \
    git \
    gnupg \
    jq \
    lsb-release \
    make \
    python3-pip \
    software-properties-common \
    sudo \
    unzip \
    wget \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install development tools
RUN set -ex \
    # Determine architecture once
    && ARCH=$(uname -m) \
    # Define architecture suffixes based on ARCH
    && if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
           GO_PLATFORM_ARCH="arm64"; \
           NODE_PLATFORM_ARCH="arm64"; \
           KUBECTL_PLATFORM_ARCH="arm64"; \
       else \
           GO_PLATFORM_ARCH="amd64"; \
           NODE_PLATFORM_ARCH="x64"; \
           KUBECTL_PLATFORM_ARCH="amd64"; \
       fi \
    # Install Go
    && echo "Installing Go for ${GO_PLATFORM_ARCH}..." \
    && curl -fsSL "https://dl.google.com/go/go${GO_VERSION}.linux-${GO_PLATFORM_ARCH}.tar.gz" -o go.tar.gz \
    && echo "${GO_SHA256}  go.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz \
    && apt-get clean \
    # Install Node.js
    && echo "Installing Node.js for ${NODE_PLATFORM_ARCH}..." \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${NODE_PLATFORM_ARCH}.tar.xz" -o node.tar.xz \
    && echo "${NODE_SHA256}  node.tar.xz" | sha256sum -c - \
    && mkdir -p /usr/local/node \
    && tar -xJf node.tar.xz -C /usr/local/node --strip-components=1 \
    && rm node.tar.xz \
    && ln -s /usr/local/node/bin/node /usr/local/bin/node \
    && ln -s /usr/local/node/bin/npm /usr/local/bin/npm \
    && ln -s /usr/local/node/bin/npx /usr/local/bin/npx \
    && apt-get clean \
    && npm install -g npm@latest \
    # Install gcloud CLI
    && echo "Installing gcloud CLI..." \
    && echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg \
    && apt-get update && apt-get install -y google-cloud-sdk \
    # Install kubectl
    && echo "Installing kubectl for ${KUBECTL_PLATFORM_ARCH}..." \
    && curl -LO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/${KUBECTL_PLATFORM_ARCH}/kubectl" \
    && echo "${KUBECTL_SHA256}  kubectl" | sha256sum -c - \
    && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
    && rm kubectl \
    && rm -rf /var/lib/apt/lists/*

FROM base
ARG USERNAME

# Create non-root user, using 'developer' if USERNAME=root
RUN TARGET_UID="1000" TARGET_GID="1000" \
    && groupadd --gid ${TARGET_GID} ${USERNAME} \
    && useradd -l --uid ${TARGET_UID} --gid ${USERNAME} --shell /bin/bash --create-home ${USERNAME} \
    && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} \
    && chmod 0440 /etc/sudoers.d/${USERNAME}

# Setup workspace and cache directories
RUN mkdir -p /workspace \
    && mkdir -p /home/${USERNAME}/.cache/go-build \
    && mkdir -p /go/pkg/mod \
    && chown -R ${USERNAME}:${USERNAME} /workspace \
    && chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}/.cache \
    && chown -R ${USERNAME}:${USERNAME} /go

# Configure npm for non-root global installs
USER $USERNAME
RUN mkdir -p /home/${USERNAME}/.npm-global \
    && npm config set prefix "/home/${USERNAME}/.npm-global"

# Environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH=/go
ENV NODE_ENV=development
ENV PATH="/usr/local/go/bin:/go/bin:${HOME}/.npm-global/bin:${PATH}"

WORKDIR /workspace

# Copy and setup .bashrc
COPY --chown=${USERNAME}:${USERNAME} .bashrc /home/${USERNAME}/.bashrc
