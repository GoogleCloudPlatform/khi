> **Note:** In this document, "I" refers to the Gemini assistant, and "you" refers to the user.

# GEMINI.md for `scripts/depguard-generator`

This document provides guidelines for the scripts located in the `scripts/depguard-generator` directory. All scripts here should follow these conventions, in addition to the global standards in the root [`GEMINI.md`](../../GEMINI.md).

## 1. Purpose of the Script

This script generates the `.generated-golangci-depguard.yaml` file.
This configuration file is used by the `golangci-lint` tool to enforce dependency rules within the Go source code. It helps maintain the project's architecture by preventing packages from being imported into modules where they don't belong.

For example, it defines rules to ensure that the `pkg/common` package does not depend on any other project-specific packages.

## 2. How to Run

To generate or update the configuration file, run the following command from the project root directory:

```bash
make generate-depguard-rules
```
