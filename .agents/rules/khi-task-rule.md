---
trigger: glob
globs: pkg/task/**/*.go
---

# KHI Task Standards

When developing or modifying task-related files in the KHI project (under `pkg/task/`), you **must** adhere to the following rules and best practices based on the KHI Task System Concept.

## Package folders

- Each task package under `pkg/task/inspection/<domain>/<feature>` follows this layout:
  - The root package (`package <feature>`) defines `TaskID`s (`taskid.go`), `FieldSet`s, `Extractor` interfaces, or other shared types used across packages.
  - The `impl/` subpackage (`package <feature>_impl`) defines the actual task implementations and exports `var Module = coreinspection.Module{...}` in `module.go`.
  - An optional `internal/` subpackage (`package <feature>_internal`) defines utilities used only within the root or `impl` packages.
- Add a `README.md` (and `README_ja.md`) directly under the task package summarizing the tasks defined in the package and their structure.

## 2. Inputs and Result Retrieval

- Tasks declare their inputs during bind time on `*coretask.Binder` using `coretask.Use(b, Reference.Ref())`, `coretask.UseOptional(b, Reference.Ref(coretask.FromActiveGraph))`, `coretask.UseTag(b, Tag.Ref())`, or `coretask.After(b, dep)`.
- Values output by upstream tasks must be retrieved at execution time by calling `.Get(ctx)` on the handle returned by `Use`, `UseOptional`, or `UseTag`, passing the context passed to the task function.

## 3. Logging

- **MUST USE** context-aware logging methods such as `slog.InfoContext`, `slog.WarnContext`, or `slog.ErrorContext`.
- Do not use non-context-aware counterparts like `slog.Info` or `fmt.Printf` within tasks.

## 4. Testing Tasks

- Generate a context for test from `inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())`
- Run the tested Task with `inspectiontest.Run` (or `tasktest.Run` for core tasks) using `tasktest.Given` / `tasktest.GivenTag`.
- When you test result of a Task emits ChangeSet, use `testchangeset.ChangeSetAsserter` and its implementation to test.
- Search existing codes for reference.

## 5. Proposing an implementation plan

- When you propose an implementation plan to user, include the expected task graph in Mermaid format.

## 6. Calling Google Cloud APIs

- When calling Google Cloud APIs (Cloud Logging, Cloud Monitoring, etc.), you **MUST** inject call options into `context.Context` using `CallOptionInjector.InjectToCallContext(ctx, container)` before executing the API call. Refer to the `googlecloud-api` skill for details and patterns.
- Tasks calling Google Cloud APIs must bind `gcpcommon.APIClientCallOptionsInjectorTaskID.Ref()` via `coretask.Use(b, ...)` (or `coretask.UseOptional(b, ...)` when the injector is optional) and read it with `.Get(ctx)`.
