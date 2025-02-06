# Development Container Setup

This directory contains configuration for a containerized development environment that provides consistent tooling and dependencies

## Quick Start

### Using VS Code

1. Install Docker and VS Code with Dev Containers extension
2. Open the project in VS Code
3. Click "Reopen in Container" when prompted

### Using Other IDEs/Editors

1. Build and start the container:

```bash
docker compose -f .devcontainer/docker-compose.yml up -d
```

2. Enter the container and run setup script:

```bash
docker compose -f .devcontainer/docker-compose.yml exec dev-env bash
/workspace/.devcontainer/setup-dev.sh
```

## Verifying Setup

```bash
# Core tools
go version
node --version
npm --version

# Development tools
goimports -h
ng --version

# Cloud tools
gcloud --version
kubectl version --client
```

