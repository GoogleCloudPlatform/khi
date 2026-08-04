# KHI Task System Syntax and Execution Modes

[< Back to Index](../khi-task-system-concept.md) | [Next: Log Processing Cookbook >](./02-log-processing-cookbook.md)

---

This document explains the **basic syntax, execution lifecycle modes (`Run` and `DryRun`), and testing methods** required to understand and develop tasks in KHI.

## 1. Directed Acyclic Graph (DAG) Basics

A DAG (Directed Acyclic Graph) is a graph that flows in one direction without any cycles. In KHI, this represents a workflow where tasks run in a specific order based on their dependencies. Each node in the graph is a task, and each edge represents a dependency between tasks.

## 2. Task Types (`Task[T]`)

Every task in KHI has an associated "type" for its output. These are written using Go generics and verified at compile time.
The following example shows how to declare a task that returns an `int` value:

```go
var IntGeneratorTask = task.NewTask(
    IntGeneratorTaskID,
    []taskid.UntypedTaskReference{},
    func(ctx context.Context) (int, error) {
        return 1, nil
    },
)
```

In this example, the following key elements are declared:

1. **`IntGeneratorTask` type**: The Go compiler infers the task type as `task.Task[int]` using generic inference.
2. **First argument (`IntGeneratorTaskID`)**: This indicates the ID of the task implementation in the task graph. It must have the type `taskid.TaskImplementationID[int]`. You can use this ID to reference the task from other tasks.
3. **Second argument (`[]taskid.UntypedTaskReference`)**: This is the list of task references that this task depends on. It is used to order the task graph and read values from other tasks in the graph.
4. **Third argument (execution function)**: The return value of this function must match the task's type parameter (`int` in this case), and it must always return an error as the second return value.

## 3. Reading Values from Tasks

To read a value from a task, you must include a reference to that target task in the dependency list.
This allows the task execution function to safely get the return value from the dependency task using `coretask.GetTaskResult(ctx, dependencyTaskRef)`:

```go
var DoubleIntTask = task.NewTask(
    DoubleIntTaskID,
    []taskid.UntypedTaskReference{IntGeneratorTaskID.Ref()}, // Specify the dependency task reference
    func(ctx context.Context) (int, error) {
        // Pass the context and reference ID to get the result
        value := coretask.GetTaskResult(ctx, IntGeneratorTaskID.Ref())
        return value * 2, nil
    },
)
```

> [!IMPORTANT]
> **Do not access undeclared dependencies**
> Calling `coretask.GetTaskResult` for a task that is not declared in the dependency list causes a runtime panic. Always include the target task reference in the dependency list passed as the second argument.

## 4. Logging Inside Tasks (`slog`)

When debugging tasks or analyzing errors, use `slog` (a structured logger with context, such as `slog.InfoContext` or `slog.ErrorContext`) instead of standard `fmt.Println`.

```go
slog.InfoContext(ctx, "processing int value", "intValue", value)
```

This automatically includes the inspection trace ID and execution context information in the log message, making it easy to trace logs in Cloud Logging or local debugging logs.

## 5. Task Package Structure and Naming Conventions

All inspection tasks in KHI follow an architectural principle of **isolating each feature into a dedicated package**.
Define each task in its own folder under `pkg/task/inspection/<package-name>/`, and separate the contents into two distinct directories:

```text
pkg/task/inspection/<package-name>/
├── contract/  # Contains only public interfaces, task IDs, and type definitions (no implementation logic)
└── impl/      # Contains actual task definitions and log processing logic (implementation)
```

### 1. Responsibilities of `contract` Folder

- It defines only **task IDs**, **interfaces**, and **public data structures (such as structs or enums)** that the feature exposes to other tasks.
- **It must not contain any functions with actual processing logic or task definitions (`task.NewTask(...)`).**
- It is the only public layer that can be imported by any other task packages.

### 2. Responsibilities of `impl` Folder

- It contains the actual tasks (`var SomeTask = task.NewTask(...)`) and parser or mapper logic bound to the task IDs defined in `contract`.
- It contains `init()` or initialization functions (such as `Register(...)`).
- **You must not import the `impl` package of other features.** When depending on another feature, import only its `contract` package and connect via task dependencies on the task graph.

### 3. Naming Conventions for Package Names (`package` declaration)

In Go, declaring packages simply as `contract` or `impl` causes name collisions when importing multiple packages and obscures where code originates.
Therefore, Go source files under `contract` and `impl` directories must **use the parent feature name combined with `_contract` or `_impl` as the package name**:

- **Files under `contract/`**: `package <feature-name>_contract` (e.g., `package example_contract`)
- **Files under `impl/`**: `package <feature-name>_impl` (e.g., `package example_impl`)

When referencing types or IDs from other tasks, always import only this `<feature-name>_contract` package.

## 6. Inspection Task Execution Modes (`Run` and `DryRun`)

Inspection tasks in KHI are invoked in either **`Run` mode** or **`DryRun` mode** depending on the execution situation. This is a fundamental lifecycle feature in task graph execution, designed to cleanly separate heavy analysis logic from lightweight UI interactions.

- **`Run` mode (`TaskModeRun`)**: The normal execution mode used when the user clicks the "Start Inspection" button to start an analysis. It executes actual log queries, syntax parsing, and serialization to generate the history file (KHI file).
- **`DryRun` mode (`TaskModeDryRun`)**: A lightweight execution mode used when the user changes input parameters or dynamically fetches form fields and autocomplete suggestions in the "New Inspection" screen. To maintain responsive UI interactions, time-consuming log queries and parsing operations are skipped in this mode.

### Example Code for Checking Execution Mode

Every inspection task (or low-level task utility) should check `inspectioncore_contract.InspectionTaskModeType` (`taskMode`) passed as an argument and switch its behavior based on the current mode.
The following is a standard Go implementation example that returns an empty result or necessary UI metadata without heavy processing during `DryRun`, and executes actual parsing only in `Run` mode:

```go
var ExampleInspectionTask = inspectiontaskbase.NewProgressReportableInspectionTask(
    ExampleInspectionTaskID,
    []taskid.UntypedTaskReference{SourceLogsTaskID.Ref()},
    func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (ResultType, error) {
        // 1. Check DryRun mode: Return immediately to skip heavy log fetching and parsing for form setup or lightweight runs
        if taskMode == inspectioncore_contract.TaskModeDryRun {
            return ResultType{}, nil
        }

        // 2. Run mode: Perform actual log fetching and time-consuming analysis or calculation
        logs := coretask.GetTaskResult(ctx, SourceLogsTaskID.Ref())
        result, err := doHeavyAnalysis(ctx, logs, progress)
        if err != nil {
            return ResultType{}, err
        }
        return result, nil
    },
)
```

By consistently applying this "early return by mode" pattern across all tasks, KHI maintains responsive and fast interactions on the "New Inspection" screen even when configuring complex log analysis task graphs.

## 7. Task Testing

You can test individual tasks and task graphs independently using the testing utilities provided by KHI.
This section describes testing using the `tasktest` package.

### 7.1 `tasktest.RunTask`

To verify the behavior of a single task simply, you can call `tasktest.RunTask`.

```go
func TestIntGeneratorTask(t *testing.T) {
    res, err := tasktest.RunTask(t.Context(), IntGeneratorTask, map[taskid.UntypedTaskImplementationID]any{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 1 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 1, res)
    }
}
```

The third argument allows you to pass a map that mocks return values from dependency tasks.
For tasks without dependencies, pass an empty map.

### 7.2 `tasktest.RunTaskWithDependency`

When you want to verify the execution of a sub-graph including its dependencies rather than mocking them, use `tasktest.RunTaskWithDependency`.
This automatically builds, topologically sorts, and executes a mini-task graph containing the dependency tasks.

```go
func TestDoubleIntTaskWithDependency(t *testing.T) {
    res, err := tasktest.RunTaskWithDependency(t.Context(), DoubleIntTask, []coretask.UntypedTask{
        IntGeneratorTask,
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 2 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 2, res)
    }
}
```

---

[< Back to Index](../khi-task-system-concept.md) | [Next: Log Processing Cookbook >](./02-log-processing-cookbook.md)
