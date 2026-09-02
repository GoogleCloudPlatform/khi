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
* `make format-go`, `make format-web`: Runs the backend and frontend formatters respectively.
* Run `make pre-commit` before finalizing a task. It runs formatter for every file.

## Technical stack

* Go version is `1.25.x`
* Angular version is `21.x`

## Common rules

Language specific rules are written in each language's rule files. Please respect them.

* All comments must be written in English.
* License headers are automatically added by commit hook. Do not add license header.
* Do not remove test cases or weaken test assertions without asking the user. However, updating test call sites, constructor parameters, and argument types to adapt to refactored signatures is required.
* KHI is an integrated monorepo application. Internal code (Go/TS functions, types, interfaces, internal APIs, and task DAGs) does NOT require backward compatibility. When updating code, unify directly onto the new implementation and update all call sites across the monorepo without retaining legacy shims or wrappers. The only exception is `.khi` file persistence, where compatibility requirements must be discussed during planning. Refer to the `backward-compatibility-policy` skill for anti-patterns and details.
* Do not perform git commit/push without any approval from the user.
* Do not assume before reading files. Read the file before changing it.
