# Kubernetes History Inspector (KHI)

## Folder structure

* `/pkg` , `/cmd`, `/internal` are for backend (Go).
* `/web` is for frontend (Angular).
* `/scripts` is for build scripts.
* `/docs` is for documentation.

## Common commands

All the following `make` commands must be run from the root folder:

* `make build-go`, `make build-web`: Builds the backend and frontend source code respectively.
* `make test-go`, `make test-web`: Runs the backend and frontend tests respectively.
* `make lint-go`, `make lint-web`: Runs the backend and frontend linters respectively.

## Technical stack

* Go version is `1.25.x`
* Angular version is `21.x`

## Common rules

Language specific rules are written in each language's rule files. Please respect them.

* All comments must be written in English.
* License headers are automatically added by commit hook. Do not add license header.
* Do not modify/remove existing test codes without asking it to the user.
* Do not perform git commit/push without any approval from the user.
* Do not assume before reading files. Read the file before changing it.
