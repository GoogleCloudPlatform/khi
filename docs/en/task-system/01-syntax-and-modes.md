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
var IntGeneratorTask = coretask.Define(
    IntGeneratorTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (int, error) {
        return func(ctx context.Context) (int, error) {
            return 1, nil
        }
    },
)
```

In this example, the following key elements are declared:

1. **`IntGeneratorTask` type**: The Go compiler infers the task type as `coretask.Task[int]` using generic inference.
2. **First argument (`IntGeneratorTaskID`)**: This indicates the ID of the task implementation in the task graph. It must have the type `taskid.TaskImplementationID[int]`. You can use this ID to reference the task from other tasks.
3. **Second argument (`bind` function)**: At task construction time, `Define` passes a `*coretask.Binder` to this outer function so the task can declare its inputs (`coretask.Use`, `coretask.UseOptional`, `coretask.UseTag`, or `coretask.After`). The `bind` function returns the execution closure (`func(ctx context.Context) (int, error)`), whose return type must match the task's type parameter (`int` in this case). For static constant outputs, you can also use `coretask.DefineConstant(IntGeneratorTaskID, 1)`.

## 3. Reading Values from Tasks

### 3.1 Point-to-Point Dependencies (`coretask.Use`)

To read a value from an upstream task, bind its reference (`taskID.Ref()`) on `*coretask.Binder` using `coretask.Use` and read the returned `coretask.Input[T]` handle inside the execution closure:

```go
var DoubleIntTask = coretask.Define(
    DoubleIntTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (int, error) {
        // Bind the required input on the Binder
        intInput := coretask.Use(b, IntGeneratorTaskID.Ref())
        return func(ctx context.Context) (int, error) {
            // Read the value from the typed input handle
            return intInput.Get(ctx) * 2, nil
        }
    },
)
```

> [!IMPORTANT]
> **Bind all inputs before the `bind` function returns**
> The `*coretask.Binder` is sealed as soon as the outer `bind` function returns. Because values can only be read through an `Input[T]`, `OptionalInput[T]`, or `TagInput[T]` handle obtained from `b`, a task cannot accidentally read an undeclared dependency at runtime. For ordering-only dependencies where the result value is not read, register them with `coretask.After(b, dep)`.

#### Dependency Scopes

When you declare a dependency using `taskID.Ref()`, the graph resolver uses a **dependency scope** to decide how to locate and activate the target task:

- **`taskid.ScopeAll` (`coretask.FromAll`)**: Eagerly searches all registered tasks and pulls the target task into the execution graph. This is the **default for point-to-point references** (`taskID.Ref()`).
- **`taskid.ScopeActiveFeatures` (`coretask.FromActiveFeatures`)**: Pulls in producer tasks only if they belong to features that are enabled for the current inspection. This is the **default for tag fan-in references** (`tag.Ref()`).
- **`taskid.ScopeActiveGraph` (`coretask.FromActiveGraph`)**: Lazily binds only to tasks that are already included in the active graph by other dependencies. It never pulls new upstream tasks into the graph on its own.

You can override the default scope when declaring dependencies (see [3.4 Explicit Dependency Scope Specification](#34-explicit-dependency-scope-specification-coretaskfrom)).

### 3.2 Optional Dependencies (`coretask.UseOptional`)

When a dependency is not guaranteed to be active in the task graph (for example, if it belongs to an optional feature that the user may disable), bind a scoped reference (`coretask.FromActiveGraph` or `coretask.FromActiveFeatures`) using `coretask.UseOptional`:

```go
var SafeConsumerTask = coretask.Define(
    SafeConsumerTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (string, error) {
        // UseOptional requires a scope narrower than ScopeAll (such as FromActiveGraph or FromActiveFeatures)
        optInput := coretask.UseOptional(b, OptionalTaskID.Ref(coretask.FromActiveGraph))
        return func(ctx context.Context) (string, error) {
            if val, ok := optInput.Get(ctx); ok {
                return val, nil
            }
            return "fallback", nil
        }
    },
)
```

### 3.3 Tag-based Fan-In Dependencies (`coretask.UseTag`)

When multiple producers contribute items of the same type (for example, multiple log parsers producing log summaries), use a `Tag[T]` and bind it with `coretask.UseTag`:

```go
// 1. Declare a tag in the feature root package
var LogItemTag = coretask.NewTag[*LogItem]("khi.google.com/log-items")

// 2. Producer tasks declare that they provide the tag (optionally specifying WithTagPriority)
var ParserTaskA = coretask.Define(
    ParserTaskAID,
    func(b *coretask.Binder) func(ctx context.Context) (*LogItem, error) {
        sourceLogs := coretask.Use(b, SourceLogRef)
        return func(ctx context.Context) (*LogItem, error) {
            return parseLogA(ctx, sourceLogs.Get(ctx))
        }
    },
    coretask.ProvidesTag(LogItemTag, coretask.WithTagPriority(10)),
)

// 3. Consumer task aggregates all active producers with UseTag
var AggregatorTask = coretask.Define(
    AggregatorTaskID,
    func(b *coretask.Binder) func(ctx context.Context) ([]*LogItem, error) {
        itemsInput := coretask.UseTag(b, LogItemTag.Ref())
        return func(ctx context.Context) ([]*LogItem, error) {
            return itemsInput.Get(ctx), nil
        }
    },
)
```

#### Fan-In Circular Dependencies and Achieving a Stable Graph via Priority

When using fan-in aggregation, circular dependencies (cycles) can arise under the following prerequisite conditions:

1. **Prerequisite Conditions (Cross-Inventory Dependencies)**:
   When multiple independent log parsers (for example, audit log parser and container log parser) produce different inventories (such as IP address lists and container ID lists) while simultaneously consuming each other's inventories to narrow their queries. While each parser is an acyclic DAG in isolation, enabling both features simultaneously dynamically forms a mutual dependency loop via fan-in aggregations (`TagReference`).
2. **Deterministic Pruning via Priority for a Stable Graph**:
   Arbitrarily cutting edges to break cycles causes execution order and data flow to fluctuate based on task registration order, producing an unreproducible, unstable graph.
   - `coretask.WithTagPriority(priority)` (default: `DefaultTagPriority = 100`, where lower numbers indicate higher precedence) lets producers declare the certainty and priority of their contribution.
   - The graph resolver deterministically prunes candidate fan-in edges that form cycles, consistently producing a safe, unique, and stable single-stage DAG.

For architectural details, see [Concept Guide: 5. Prerequisites of Fan-In Cycles and Graph Stabilization via Priority](../khi-task-system-concept.md#5-prerequisites-of-fan-in-cycles-and-graph-stabilization-via-priority).

### 3.4 Explicit Dependency Scope Specification (`coretask.From*`)

When binding optional dependencies (`coretask.UseOptional`), ordering dependencies (`coretask.After`), or narrowing tag fan-in (`coretask.UseTag`), pass a scope option (`coretask.FromActiveGraph` or `coretask.FromActiveFeatures`) when creating the reference (`coretask.Use` always requires the default `ScopeAll`):

```go
var AdvancedConsumerTask = coretask.Define(
    AdvancedConsumerTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (ResultType, error) {
        // Lazily bind to tag producers that are already in the active graph
        itemsInput := coretask.UseTag(b, LogItemTag.Ref(coretask.FromActiveGraph))

        // Bind an optional task if its feature is enabled for the current inspection
        optInput := coretask.UseOptional(b, OptionalTaskID.Ref(coretask.FromActiveFeatures))

        return func(ctx context.Context) (ResultType, error) {
            items := itemsInput.Get(ctx)
            optVal, _ := optInput.Get(ctx)
            return computeResult(items, optVal), nil
        }
    },
)
```

## 4. Logging Inside Tasks (`slog`)

When debugging tasks or analyzing errors, use `slog` (a structured logger with context, such as `slog.InfoContext` or `slog.ErrorContext`) instead of standard `fmt.Println`.

```go
slog.InfoContext(ctx, "processing int value", "intValue", value)
```

This automatically includes the inspection trace ID and execution context information in the log message, making it easy to trace logs in Cloud Logging or local debugging logs.

## 5. Task Package Structure and Naming Conventions

All inspection tasks in KHI follow an architectural principle of **isolating each feature into a dedicated package**.
Define each task in its own folder under `pkg/task/inspection/<domain>/<feature>/`, separating the public contract at the package root from the implementation in `impl/`:

```text
pkg/task/inspection/<domain>/<feature>/
├── taskid.go  # Package <feature>: public task IDs, interfaces, extractors, and type definitions
└── impl/      # Package <feature>_impl: actual task definitions, log processing logic, and module.go
```

### 1. Responsibilities of the Feature Root Package

- It defines only **task IDs** (`taskid.go`), **interfaces**, **extractor functions**, and **public data structures (such as structs or enums)** that the feature exposes to other tasks.
- **It must not import `impl` or contain task definitions (`coretask.Define(...)`).**
- It is the public layer that can be imported by any other task packages.

### 2. Responsibilities of the `impl` Folder

- It contains the actual tasks (`var SomeTask = coretask.Define(...)` or `inspectiontaskbase.DefineInspectionTask(...)`) and parser or mapper logic bound to the task IDs defined in the root package.
- It exports `var Module = coreinspection.Module{...}` in `module.go` to declare its inspection scope and tasks.
- **You must not import the `impl` package of other features.** When depending on another feature, import only its root package and connect via task dependencies on the task graph.

### 3. Naming Conventions for Package Names (`package` declaration)

- **Files at the feature root (`pkg/task/inspection/<domain>/<feature>/`)**: `package <feature>` (e.g., `package k8snode`)
- **Files under `impl/`**: `package <feature>_impl` (e.g., `package k8snode_impl`)

When referencing types or task IDs from other tasks, always import only the `<feature>` root package.

## 6. Inspection Task Execution Modes (`Run` and `DryRun`)

Inspection tasks in KHI are invoked in either **`Run` mode** or **`DryRun` mode** depending on the execution situation. This is a fundamental lifecycle feature in task graph execution, designed to cleanly separate heavy analysis logic from lightweight UI interactions.

- **`Run` mode (`TaskModeRun`)**: The normal execution mode used when the user clicks the "Start Inspection" button to start an analysis. It executes actual log queries, syntax parsing, and serialization to generate the history file (KHI file).
- **`DryRun` mode (`TaskModeDryRun`)**: A lightweight execution mode used when the user changes input parameters or dynamically fetches form fields and autocomplete suggestions in the "New Inspection" screen. To maintain responsive UI interactions, time-consuming log queries and parsing operations are skipped in this mode.

### Example Code for Checking Execution Mode

Every inspection task (or low-level task utility) should check `inspectioncore.InspectionTaskModeType` (`taskMode`) passed as an argument and switch its behavior based on the current mode.
The following is a standard Go implementation example that returns an empty result or necessary UI metadata without heavy processing during `DryRun`, and executes actual parsing only in `Run` mode:

```go
var ExampleInspectionTask = inspectiontaskbase.DefineInspectionTask(
    ExampleInspectionTaskID,
    func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[ResultType] {
        logsInput := coretask.Use(b, SourceLogsTaskID.Ref())
        return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (ResultType, error) {
            // 1. Check DryRun mode: Return immediately to skip heavy log fetching and parsing for form setup or lightweight runs
            if taskMode == inspectioncore.TaskModeDryRun {
                return ResultType{}, nil
            }

            // 2. Run mode: Perform actual log fetching and time-consuming analysis or calculation
            result, err := doHeavyAnalysis(ctx, logsInput.Get(ctx))
            if err != nil {
                return ResultType{}, err
            }
            return result, nil
        }
    },
    progress.WithTitle("Analyze source logs"),
)
```

By consistently applying this "early return by mode" pattern across all tasks, KHI maintains responsive and fast interactions on the "New Inspection" screen even when configuring complex log analysis task graphs.

## 7. Task Testing

You can test individual tasks and task graphs independently using the testing utilities provided by KHI.
This section describes testing using the `tasktest` package.

### 7.1 `tasktest.Run`

To verify the behavior of a single task in isolation, call `tasktest.Run` and provide upstream input values with `tasktest.Given` (for point-to-point references) or `tasktest.GivenTag` (for tag fan-in references):

```go
func TestIntGeneratorTask(t *testing.T) {
    res, err := tasktest.Run(t, t.Context(), IntGeneratorTask)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 1 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 1, res)
    }
}

func TestDoubleIntTask(t *testing.T) {
    res, err := tasktest.Run(
        t,
        t.Context(),
        DoubleIntTask,
        tasktest.Given(IntGeneratorTaskID.Ref(), 5),
    )
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 10 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 10, res)
    }
}
```

`tasktest.Run` validates that every required input declared on the task's `Binder` is supplied via `tasktest.Given` and that no undeclared inputs are passed.

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
