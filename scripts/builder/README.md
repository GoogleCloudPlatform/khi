# KHI CI Builder Image

This directory contains the Dockerfile for building the unified CI builder image used in `.cloudbuild/` workflows.

## Included Dependencies

- Debian Trixie (base OS)
- Go 1.26.x (copied from `golang:1.26-trixie`)
- Node.js 26.x, npm, npx (base from `node:26-trixie`)
- System utilities: `jq`, `make`, `git`, `curl`

## Building and Pushing

You can build and push this image using the project `Makefile`:

```bash
# Build the image locally
make build-builder

# Build and push the image to GCR
make push-builder
```
