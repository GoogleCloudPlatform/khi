> **Note:** In this document, "I" refers to the Gemini assistant, and "you" refers to the user.

# GEMINI.md for `scripts` Directory

This document provides guidelines for the scripts located in the `scripts/` directory. All scripts here should follow these conventions, in addition to the global standards in the root [`GEMINI.md`](../GEMINI.md).

## 1. Purpose and Usage of Scripts

The `scripts/` directory contains shell scripts and other automation tools used for development, building, and maintenance tasks. The primary goal is to automate repetitive tasks and ensure consistency.

### Key Scripts

- **`generate-angular-json.sh`**: Generates or updates the `angular.json` file based on project templates. This should be run when frontend dependencies or configurations change.
- **`generate-version.sh`**: Creates the `VERSION` file, which is used to tag builds and releases.
- **`pre-commit`**: This is the git pre-commit hook script. It runs linters and formatters before a commit is made to ensure code quality. It is set up via `make setup-hooks`.
- **`frontend-codegen/`**: Contains scripts related to frontend code generation.
- **`make/`**: Contains scripts that are primarily invoked by the root `Makefile`.

## 2. Development Guidelines

When adding or modifying scripts, please adhere to the following rules.

### General Rules

- **Shell**: Write all scripts in **`bash`** for maximum portability. Avoid using features specific to other shells like `zsh`. Start scripts with `#!/bin/bash`.
- **Permissions**: Ensure any new script is executable by running `chmod +x your-script-name.sh`.
- **Dependencies**: If a script requires a command-line tool (like `jq`, `yq`, etc.), document it clearly at the top of the script file.
- **Error Handling**: Use `set -e` at the beginning of your scripts to ensure they exit immediately if a command fails. Check for unbound variables with `set -u`.

### Naming Conventions

- **Variables**: Use `UPPER_CASE` for environment variables and global constants. Use `lower_case` for local variables.
- **Functions**: Use `lower_case_with_underscores()` for function names.

## 3. Integration with Makefile

Most scripts in this directory are designed to be called from the root `Makefile`, which serves as the single entry point for all common developer tasks.

- **Define Makefile Targets**: When adding a new process, consider adding a new target for it in the `Makefile`.
- **Call Scripts from Makefile**: If a task requires a long shell command or script, it should be placed in a separate script file in this directory and invoked from the `Makefile`.
