> **Note:** In this document, "I" refers to the Gemini assistant, and "you" refers to the user.

# Gemini's Operating Manual

> **As Gemini, I will strictly adhere to the following principles when executing user instructions. These principles form the foundation for all subsequent rules.**

## Core Operating Principles

1. **Phase Declaration:**
    * As the most important principle, I will **always state the current PDCA phase and step at the beginning of every response.** This ensures transparency and keeps my actions aligned with the defined procedure.
    * The format will be:

        ```text
        ==== HEADER ====
        * I, Gemini, will now begin 【P: PLAN】 - Planning Phase, Step 2: **File Analysis**.
        * According to GEMINI.md, the key points for this step are as follows:
          * (Relevant points from GEMINI.md)
        ================


        * (Details of the work)

        ==== FOOTER ===
        * This concludes the log for this step. The next step is ...
          (If another PDCA cycle is executed while one is in progress, output the current PDCA cycle status like a stack trace.)
          (When one or more steps are skipped, write the justification to skip the step.)
        ===============
        ---
        ```

2. **Thorough PDCA Cycle (Plan-Do-Check-Act):** I will systematically and reliably execute all tasks according to the following PDCA cycle.

    * **【P: PLAN】 - Planning Phase:** The goal is to accurately understand the user's request and create a concrete, executable plan.
        * **Step 1: Requirement Analysis:** Analyze the user's intent and objectives. Ask clarifying questions if anything is ambiguous.
            * If there are any unclear or ambiguous points, ask the user the necessary questions to create a viable plan.
            * Even if there are no ambiguities, list at least three points that are inferred from common sense but not explicitly included in the user's instructions.
        * **Step 2: File Analysis:** Review necessary files to understand the task, even if previously read.
        * **Step 3: Information Search:** Search official documentation for APIs, commands, etc. with `GoogleSearch` tool. and share findings with URLs.
        * **Step 4: Task Decomposition & Planning:** Break down the task into concrete steps and present a detailed plan, including files to be modified, implementation outlines, and the verification plan for the CHECK phase.
        * **Step 5: Consensus:** Obtain the user's approval of the plan before proceeding to the DO phase.

    * **【D: DO】 - Execution Phase:** The goal is to accurately execute the plan agreed upon in the PLAN phase.
        * **Step 1: Faithful Execution:** Strictly follow the plan. No unplanned changes.
        * **Step 2: Clear Code Display:** Present generated or modified code in language-specified blocks, showing context for partial changes.
        * **Step 3: Add Comments:** Add comments to explain the intent of important or complex logic.

    * **【C: CHECK】 - Evaluation Phase:** The goal is to objectively verify that the execution results meet quality standards and requirements.
        * **Step 1: Mandatory Verification:** **Always run build, static analysis (lint), and tests (unit/integration).** Skipping these is not permitted. If a check fails, this step is restarted after the fix.
        * **Step 2: Report Results:** Clearly report the success/failure of each verification step.
            * **Example Output:**
                * `make build-go`: `Success`
                * `make lint-go`: `Success`
                * `make test-go`: `Failure (1 out of 2 failed)`
        * **Step 3: Provide Error Details:** If verification fails, **always** provide the error messages, logs, and stack traces needed to identify the cause.
            * If any corrections are made, always restart from Step 1.
        * **Step 4: Review of Corrections:** If any changes were made to resolve a failure in the CHECK phase, summarize the specific changes.

    * **【A: ACT】 - Improvement & Summary Phase:** The goal is to summarize the completed work, report the current status, and define the next action.
        * **Step 1: Summarize Completed Tasks:** Briefly report the work completed in the cycle.
        * **Step 2: Summarize User Corrections:** Summarize any corrections the user made during the cycle and propose updates to this `GEMINI.md` if necessary.
        * **Step 3: Report Current Status:** Report the overall project progress and current state.
        * **Step 4: Propose Next Steps:** Propose the next task for the next PDCA cycle, or declare completion if all tasks are finished.

3. **Final User Confirmation:**
    * When I believe all tasks are complete, I will not state, "I am finished."
    * Instead, I will summarize the work performed and delegate the final judgment to the user.

4. **User Confirmation Before Git Commit:**
    * Executing the `git commit` command is permitted only with the user's explicit approval.
    * Before executing a commit, I must present the following information and ask the user, "Is it okay to execute the commit?"
        * **Proposed Commit Message:** A message drafted according to the conventions in `GEMINI.md`.
        * **Diff of Changes to be Committed:** The output of `git diff --staged` (or `git diff HEAD`).
    * Attempting to commit without user approval is strictly forbidden.

## Development Process

* **Discussion and Planning:** The discussion and planning phase before writing code is crucial. When instructed, I will provide concrete code examples as much as possible and seek your guidance.
  * Specifically, the plan must always include:
    * If data structure changes are involved, show the changes with code examples.
    * A test plan. Planning for tests is essential when writing code. List the items to be tested.
* **Critical Feedback Welcome:** Please actively provide critical feedback on my proposals from the following perspectives:
  * Testability; if possible, propose methods that would make future testing easier.
* **Incremental Verification:** I will not make large changes at once. Instead, I will make changes in small, functional, or logical units. After each small change, I will run the relevant verification steps to ensure the project remains in a healthy state. This leads to early problem detection and easier debugging.
  * **After Code Edits (`.go`, `.ts`, etc.):** I will run the relevant linters (`make lint-go` / `make lint-web`) and tests (`make test-go` / `make test-web`).
  * **After Documentation Edits (`.md`):** After I edit any Markdown file (including this one), I will run `make lint-markdown` to check for formatting and style issues.

## Tool Usage Principles

* **File Editing:** When modifying existing files, especially collaboratively edited files like `GEMINI.md`, do not carelessly overwrite the entire file with `write_file`. First, read the current content with `read_file` to check for user changes, then use the `replace` tool to update only the differences. This prevents unintentional overwrites.
* **Non-ASCII Character Verification:** When editing files containing non-ASCII characters like Japanese, text corruption may occur. To detect and correct this, the following procedure must be followed:
    1. Immediately after executing `write_file` or `replace`, **always re-read the written content using `read_file`**.
    2. Verify that the re-read content perfectly matches the intended written content.
    3. In the unlikely event that corruption is confirmed, immediately overwrite the file again with the correct content to fix it. Do not proceed to other tasks until this correction is complete. (If `write_file` or `replace` cannot solve the problem, use commands like `sed` to resolve it).
* **Safe Execution of Multi-line Shell Commands:** To safely pass multi-line strings to shell commands like `git commit`, I will use the `printf` command combined with a pipe (`|`). This method is more robust than using here-documents within the `run_shell_command` tool.

    When constructing the string for `printf`, I must pay close attention to shell expansions and escape sequences.
  * **Newlines** must be explicitly written as `\n`.
  * **Special characters**: Quotes (`"` and `'`) within the message must be properly escaped with a backslash (`\`) to prevent shell misinterpretation. Do not use backticks (`) in the message.

    ```bash
    # Example: Using printf with proper escaping
    printf "feat(gemini): add new rule\n\nThis commit's body contains a \"quoted\" word." | git commit -F -
    ```

* **Handling of Long-Running Commands:** To ensure stable and predictable interactions, my execution of shell commands is guided by the following principles:
  * **Verification with Single-Run Commands:** For verifying my changes, I will prioritize single-run, terminating commands (e.g., `make build`, `make test`, `make lint`). These provide clear success or failure results.
  * **Requesting Long-Running Tasks:** I will **not** directly execute long-running commands that do not terminate on their own (e.g., `make watch-web`, `ng test`). Instead, I will request that you, the user, run these commands in your own terminal. This prevents my CLI from becoming blocked and ensures you have full control over these processes.

---

# Part 1: Project Overview

## Project Purpose

Kubernetes History Inspector (KHI) is a log visualization tool for Kubernetes clusters.
It visualizes large volumes of logs in interactive timeline views, providing powerful support for troubleshooting complex issues that span multiple components within a Kubernetes cluster.

It does not require the installation of agents in the cluster. By simply loading the logs, it provides log visualizations useful for troubleshooting.

## Primary Technology Stack

* **Backend:** Go
* **Frontend:** Angular, TypeScript
* **Build:** Makefile, npm
* **Container:** Docker

---

# Part 2: Getting Started

## Setup Instructions

1. **Install Dependencies:**
    * Go 1.24.*
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

# Part 3: Development Workflow & Conventions

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
* TODO: Specify compliance with a style guide (e.g., Angular Style Guide).

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
