# GEMINI.md for `pkg` Directory

This document outlines the development conventions and guidelines specific to the `pkg` directory. All code within this directory must adhere to these rules, in addition to the global standards defined in the root [`GEMINI.md`](../GEMINI.md).

## 1. Package Responsibilities

The `pkg` directory contains the core business logic of the Kubernetes History Inspector, written in Go. It is organized into several packages, each with a distinct responsibility.

### Current Structure

The following is the current, high-level structure:

- **`common`**: Provides generic, reusable components and utilities that are not specific to this application's domain (e.g., collections, concurrent data structures, time helpers). **This package must not depend on any other `pkg/` packages.**
- **`core`**: Contains the fundamental components of KHI's task system.
  - **`task`**: Defines the generic Directed Acyclic Graph (DAG) mechanism, independent of specific tasks. It is only allowed to depend on `pkg/common`.
  - **`inspection`**: Manages the task graph for log inspection. It is only allowed to depend on `pkg/common` and `pkg/core/task`.
  - **`app`**: Manages the task graph for the KHI application lifecycle. It is only allowed to depend on `pkg/common` and `pkg/core/task`.
- **`task`**: Contains all packages that define the concrete DAGs for KHI.
- **`testutil`**: Contains helper packages for KHI's tests. All package names under this directory must end with `_test`. These are utilities; the tests themselves are located in `_test.go` files within the same folder as the code they test. **Code outside of the `testutil` directory is forbidden from importing these packages.**
  - **`task`**: Provides testing utilities for the generic task graph.
  - **`inspection`**: Provides testing utilities for the log inspection task components.
  - **`app`**: Provides testing utilities for the application lifecycle task components.
- **`model`**: Contains the primary data models for Kubernetes objects, historical events, and their compositions.
- **`server`**: Contains the HTTP server implementation.

## 2. Coding Conventions

We follow Google's Go coding standards and the conventions outlined in the root `GEMINI.md`. The following rules are specific to the `pkg` directory.

### Error Handling

- **Error Wrapping**: All errors returned from external libraries or other packages should be wrapped with additional context using `fmt.Errorf("...: %w", err)`. This provides a clear error trace.
- **Error Reporting**: For user-facing errors or significant internal issues, use the `errorreport` package to create structured error reports.

### Logging

- **Structured Logging**: All logging must be done using the standard `slog` package to ensure a consistent, structured format.
- **Use Context**: Use context-aware logging functions (e.g., `slog.WarnContext()`) whenever possible.
- **Log Throttling**: If a high volume of similar logs is anticipated, assign a `LogKind` (currently defined in `inspection/logger/`). This will throttle the output if too many similar logs are generated.

### Naming Conventions

- **Interfaces**: Interface names should end with `-er` or `-or` (e.g., `Reader`, `Inspector`) or be named to reflect their purpose without a specific suffix if the implementation is not important to the caller.
- **Structs**: Structs that implement an interface should be named logically. For example, the implementation for a `Reader` interface might be `fileReader` or `gcsReader`.
- **Packages**:
  - Packages under `pkg/task/inspection` should follow the naming convention `[provider][resource_type]` (e.g., `googlecloudclustergke`, `ossclusterk8s`).
    - `resource_type` should be `log[log_type]` if the task is not specific to cluster type but commonly used by multiple cluster types.
    - `resource_type` should be `cluster[cluster_type]` if the task is specific to cluster type.
    - Folders directly under `pkg/task/inspection` contians `impl` and `contract` folders.
    - `impl` folder should contain the implementation of the task.
    - `contract` folder should contain the contract of the task(Task ID and types used as results in the task).
      - `contract` folder should not depend on `impl` folder.
    - `contract` folder package name is `[provider][resource_type]_contract`.
    - `impl` folder package name is `[provider][resource_type]_impl`.
- **Task Implementation**:
  - Task implementation files should end with `_task.go`.
  - Task IDs should be defined in `contract/taskid.go`.

## 3. Testing Strategy

- **Unit Tests**: All public functions and significant internal logic must be covered by unit tests. Test files should be named `_test.go`.
- **Avoid Assertion Libraries**: Do not use third-party assertion libraries. Check conditions using simple `if` statements and report test failures with standard functions like `t.Errorf()` or `t.Fatalf()`.
  - **Complex Struct Comparison**: Use `cmp.Diff` from `github.com/google/go-cmp/cmp` when comparing complex structs.
- **Prefer Table-Driven Tests**: Structure tests as table-driven tests. Define a test case struct within the test function and iterate over a slice of test cases, calling `t.Run()` for each one.
  - **ChangeSet Comparison**: When testing `history.ChangeSet`, use `testchangeset.ChangeSetAsserter` and its implementations (e.g., `HasRevision`, `HasEvent`) from `pkg/testutil/testchangeset`.
- **Test Utilities**: Use the `testutil` package for common test setup and helper functions. Avoid duplicating test logic.
  - **Task Testing**: Use `tasktest` and `inspectiontest` packages for testing tasks. See `pkg/task/inspection/googlecloudclustergke/impl/autocompletegkeclusternames_task_test.go` for a reference implementation.
- **Mocks**: When testing interactions between packages, use interfaces and mock implementations.
- **Skipping Tests**: For tests that require external dependencies (like Cloud Logging), use the `-skip-cloud-logging=true` flag as documented in the root `GEMINI.md`. Ensure such tests are properly tagged.

    ```bash
    go test ./... -args -skip-cloud-logging=true
    ```

## 4. Dependency Management

- **Adding Dependencies**: Before adding a new external dependency, I must ask you for approval. I will provide detailed information about the dependency, which I will find by searching for it on Google.
- **Updating Dependencies**: Run `go mod tidy` to ensure the `go.mod` and `go.sum` files are clean and accurate after making changes.
