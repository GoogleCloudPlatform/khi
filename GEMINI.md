# Kubernetes History Inspector (KHI) Developer Guide

## Part 1: Project Overview

## Project Purpose

Kubernetes History Inspector (KHI) is a log visualization tool for Kubernetes clusters.
It visualizes large volumes of logs in interactive timeline views, providing powerful support for troubleshooting complex issues that span multiple components within a Kubernetes By simply loading the logs, it provides log visualizations useful for troubleshooting.

## Primary Technology Stack

* **Backend:** Go
* **Frontend:** Angular 20, TypeScript
* **Build:** Makefile, npm
* **Container:** Docker

---

## Part 2: Getting Started

## Setup Instructions

1. **Install Dependencies:**
    * Go 1.25.*
    * Node.js 22.13.*
    * `gcloud` CLI
    * `jq`
2. **Clone Repository and Initial Setup:**

    ```bash
    git clone https://github.com/GoogleCloudPlatform/khi.git
    cd khi
    cd ./web && npm install
    ```

3. **Set up Git Hooks:**

    ```bash
    make setup-hooks
    ```

For more details, please refer to the [Development Guide](/docs/en/development-contribution/development-guide.md).

## Common Commands

Note that all `make` commands must be run from the root directory.

### Common

| Command | Description |
| :--- | :--- |
| `make setup-hooks` | Sets up the Git pre-commit hook. Run this once before starting development. |

### All (Backend + Frontend)

| Command | Description |
| :--- | :--- |
| `make build` | Builds all source code. |
| `make test` | Runs all tests. |
| `make lint` | Runs all linters. |
| `make format` | Formats all source code. |
| `make coverage` | Measures test coverage for all code. |

### Backend (Go)

| Command | Description |
| :--- | :--- |
| `make build-go` | Builds the backend source code. |
| `make test-go` | Runs backend tests. |
| `make lint-go` | Runs the backend linter. |
| `make format-go` | Formats the backend source code. |
| `make coverage-go` | Measures backend test coverage. |

### Frontend (Angular)

| Command | Description |
| :--- | :--- |
| `make build-web` | Builds the frontend source code for production. |
| `make watch-web` | Starts the frontend development server (<http://localhost:4200>). |
| `make test-web` | Runs frontend tests. |
| `make lint-web` | Runs the frontend linter. |
| `make format-web` | Formats the frontend source code. |
| `make coverage-web` | Measures frontend test coverage. |

### Other

| Command | Description |
| :--- | :--- |
| `make add-licenses` | Adds missing license headers to files. |
| `make lint-markdown` | Runs the linter for documentation (Markdown). |
| `make lint-markdown-fix` | Auto-fixes linter errors in documentation. |

## Debugging

Backend debugging is possible with VSCode. Please configure `.vscode/launch.json`.
For more details, refer to the [Development Guide](/docs/en/development-contribution/development-guide.md).

The frontend development server is started with `make watch-web`.

---

## Part 3: Development Workflow & Conventions

## Coding Conventions

Please follow Google's coding conventions as much as possible.

### Golang (Backend)

* If a license header is missing, add it by running `make add-licenses`. Do not try to generate the license field when creating new files.
* Add godoc-style comments to public types and their members.
* All comments must be written in English.
* Apply `gofmt` formatting (run `make format-go`).

### TypeScript (Frontend)

* If a license header is missing, add it by running `make add-licenses`. Do not try to generate the license field when creating new files.
* Add TSDoc-style comments to public types and their members.
* All comments must be written in English.
* Component selectors should have a `khi-` prefix.
* Apply `prettier` and `stylelint` formatting (run `make format-web`).
* **When creating new components or refactoring, actively follow the latest Angular syntax.**
  * **Standalone Components are the default.**
  * Use the **`input()`** signal function for component inputs instead of the `@Input` decorator.
  * Prefer **Signals** over RxJS for component-level state management.
  * In templates, use built-in control flow (**`@for`**, **`@if`**) over structural directives (`*ngFor`, `*ngIf`).

### Sass (SCSS)

* If a license header is missing, add it by running `make add-licenses`.
* Format code according to `prettier` and `stylelint` (run `make format-web`).
* Use `//` for comments and write them in English.
* **Naming Convention:** BEM (Block, Element, Modifier) is recommended for component-scoped styles (e.g., `.khi-button`, `.khi-button__icon`, `.khi-button--primary`).
* **Nesting:** To maintain readability and low specificity, selector nesting should be limited to **3 levels** as a general rule.
* **`@use` vs. `@import`:** For loading external files, use the modern **`@use`** instead of the older `@import` to prevent global namespace pollution.
* **Variables & Mixins:** Colors, font sizes, media queries, etc., that are used in multiple places should be extracted into dedicated files (e.g., `_variables.scss`, `_mixins.scss`) and loaded with `@use`.

### GLSL (Shaders)

* **Version:** The first line of a shader must always be `#version 300 es`.
* If a license header is missing, add it by running `make add-licenses`.
* Use `//` for comments and write them in English.
* **Performance:**
  * **Precision:** Start with a declaration like `precision highp float;`. Since precision is important in this application, `highp` is fine unless otherwise instructed.
  * **Branching:** `if` statements can impact performance. Whenever possible, consider expressing logic with built-in functions like `step()`, `mix()`, and `clamp()`.
* **Naming Convention (WebGL 2.0 / GLSL ES 3.00):**
  * **Vertex Shader Inputs:** Use `in` (e.g., `in vec3 a_position;`). Do not use `attribute`.
  * **Between Vertex -> Fragment:** Use `out` on the Vertex Shader side and `in` on the Fragment Shader side (e.g., `out vec2 v_uv;` / `in vec2 v_uv;`). Do not use `varying`.
  * **Uniform Variables:** Use `uniform` (e.g., `uniform mat4 u_projectionMatrix;`).
  * **Fragment Shader Output:** Use `out` (e.g., `out vec4 fragColor;`). Do not use `gl_FragColor`.
* **Magic Numbers:** Avoid writing literal numbers directly in shader code (magic numbers). Define them as `const` constants or `uniform` variables instead.

## Testing

You can run frontend and backend tests with the following command:

```bash
make test
```

To run backend tests while skipping those that use Cloud Logging:

```bash
go test ./... -args -skip-cloud-logging=true
```

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.
This makes the change history readable and enables automated release note generation.

### Format

```markdown
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

### Elements

* **type (required):** A keyword for the type of commit.
  * `feat`: A new feature
  * `fix`: A bug fix
  * `docs`: Documentation only changes.
  * `style`: Changes that do not affect the meaning of the code (formatting, etc.).
  * `refactor`: A code change that neither fixes a bug nor adds a feature.
  * `perf`: A code change that improves performance.
  * `test`: Adding missing tests or correcting existing tests.
  * `build`: Changes that affect the build system or external dependencies (e.g., `Makefile`, `package.json`).
  * `ci`: Changes to our CI configuration files and scripts (e.g., `.github/workflows/`).
  * `chore`: Other changes that don't modify src or test files.

* **scope (optional):** The scope of the commit's impact.
  * Examples: `web`, `api`, `auth`, `deps`, `docs`, `release`

* **subject (required):** A concise description of the change.
  * Use 50 characters or less.
  * Use the imperative mood (e.g., "add," "change").
  * Do not capitalize the first letter.
  * Do not end with a period.

* **body (optional):** A more detailed explanation, including the motivation for the change.
  * Explain "why" the change was made.
  * Wrap at 72 characters.

* **footer (optional):** Contains information about breaking changes and references to issues.
  * **Breaking Changes:** Start with `BREAKING CHANGE:` and explain the change and migration path.
  * **Issue References:** Use `Fixes #123`, `Closes #456`, etc.

### Examples

**Simple Fix:**

```bash
fix(web): correct display issue with login button
```

**New Feature (with details):**

```bash
feat(api): add endpoint for updating user profiles

Implements PUT /api/v1/users/{id}/profile to allow users to update their display name and bio.
The previous implementation only allowed for profile creation.

Closes #78
```

**Refactoring with a Breaking Change:**

```bash
refactor(auth): replace JWT library

The existing JWT library was deprecated and had security concerns.
Replaced with `golang-jwt/jwt`.

BREAKING CHANGE: The JWT signing algorithm has been changed from RS256 to ES256.
All clients must be updated to verify tokens with the new public key.
```
