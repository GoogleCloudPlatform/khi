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
* Run `make pre-commit` before finalizing a task. It auto-formats and runs fast static checks on modified files.
* Run `make pre-push` before pushing commits or opening a PR. It runs full CI-parity verification on changed branch files.

## Technical stack

* Go version is `1.27.x`
* Angular version is `21.x`

## Common rules

Language specific rules are written in each language's rule files. Please respect them.

* All comments must be written in English.
* Newly added source and config files require an Apache-2.0 Google LLC license header; run `make pre-commit` (or `make add-licenses`) to add headers and `make qa-fast-must` to verify before pushing with `jj`.
* Do not remove test cases or weaken test assertions without asking the user. However, updating test call sites, constructor parameters, and argument types to adapt to refactored signatures is required.
* KHI is an integrated monorepo application. Internal code (Go/TS functions, types, interfaces, internal APIs, and task DAGs) does NOT require backward compatibility. When updating code, unify directly onto the new implementation and update all call sites across the monorepo without retaining legacy shims or wrappers. The only exception is `.khi` file persistence, where compatibility requirements must be discussed during planning. Refer to the `backward-compatibility-policy` skill for anti-patterns and details.
* Do not assume before reading files. Read the file before changing it.
* When acting as Coordinator or top-level task lead in an active swarm session for multi-file refactoring, multi-package changes, or code-quality overhauls, read and follow `.agents/skills/khi_swarm_workflow/SKILL.md` (`khi-swarm-workflow`) to orchestrate Planner, Scout, Implementer, Auditor, Verifier, and Walkthrough Reviewer roles instead of working solo (keep small single-file edits direct; already-scoped role workers follow their assigned brief).
