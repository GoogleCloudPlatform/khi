---
trigger: glob
globs: pkg/task/**/*.go
---

# KHI Task Standards

When developing or modifying task-related files in the KHI project (under `pkg/task/`), you **must** adhere to the following rules and best practices based on the KHI Task System Concept.

## Package folders

- There should be only 3 folders included in each task packages `pkg/task/inspection/<package-task-name>`.
  - `contract` folder defines TaskID, FieldSetReader or other types used for defining TaskIDs. This package must have the package name `packagetaskname_contract`.
  - impl folder defines the actual tasks. This package must have the package name packagetaskname_impl. This package must have registration.go.
  - `internal` folder defines utility only used from the contract or impl folder. The package name must be `packagetaskname_internal`.
- Add a README.md just under the task package summarizing details of tasks defined in the package and the expected structure.

## 2. Dependencies and Result Retrieval

- Values output by dependent tasks should be retrieved using `task.GetTaskResult(ctx, Reference.Ref())`.
- The context passed to `GetTaskResult` must be the exact context value passed to the task function.

## 3. Logging

- **MUST USE** context-aware logging methods such as `slog.InfoContext`, `slog.WarnContext`, or `slog.ErrorContext`.
- Do not use non-context-aware counterparts like `slog.Info` or `fmt.Printf` within tasks.

## 4. Testing Tasks

- Generate a context for test from `inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())`
- Run the tested Task with `inspectiontest.RunInspectionTask`.
- When you test result of a Task emits ChangeSet, use `testchangeset.ChangeSetAsserter` and its implementation to test.
- Search existing codes for reference.

## 5. Proposing an implementation plan

- When you propose an implementation plan to user, include the expected task graph in Mermaid format.
