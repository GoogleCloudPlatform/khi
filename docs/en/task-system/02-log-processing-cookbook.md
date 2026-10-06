# Log Processing Task Implementation Patterns (Cookbook)

[< Previous: Syntax and Modes](./01-syntax-and-modes.md) | [Back to Index](../khi-task-system-concept.md) | [Next: Advanced Patterns >](./03-advanced-and-form-tasks.md)

---

This document explains the overall log processing pipeline in KHI and provides **practical cookbooks for 4 high-level task utilities** that developers use to implement new log parsers and timeline mappers.

## 1. Overview of the Log Processing Pipeline

KHI uses its robust task system to perform log processing. This architecture makes KHI highly extensible (you can add new features just by creating new tasks) and allows it to fully leverage Go's concurrency.
Because log processing shares many common patterns, you should implement log processing tasks using the high-level task creation utilities provided by KHI.

KHI provides the following high-level task creation utilities to cover basic log processing use cases:

- **`DefineLogFilterTask`** : Filters logs based on conditions.
- **`DefineLogGrouperTask`** : Groups logs by specific keys (e.g., entity name, correlation ID).
- **`DefineLogIngesterTask`** : Ingests logs into the final history data, generating log-level metadata (`LogChangeSet`).
- **`DefineLogToTimelineMapperTask`** : Maps ingested logs to timeline events and resource revisions displayed on the KHI UI (`TimelineChangeSet`).

The following is an example dependency graph showing how these task types work together to parse Kubernetes node audit logs fetched from Cloud Logging and render them on the UI timeline:

```mermaid
flowchart TD
    Fetch["Log Collection/Query Task<br>(Cloud Logging, etc.)"]
    Filter["DefineLogFilterTask<br>(Filter out unwanted logs)"]
    Grouper["DefineLogGrouperTask<br>(Group by Pod name or thread)"]
    Ingester["DefineLogIngesterTask<br>(Build log change sets and styles)"]
    Mapper["DefineLogToTimelineMapperTask<br>(Map to timeline events and paths)"]

    Fetch --> Filter
    Filter --> Grouper
    Filter --> Ingester
    Grouper --> Mapper
    Ingester --> Mapper
```

When developers add support for a new log type, they declare tasks along this pipeline and connect them to the task graph.
You create all of these using utility functions in the `pkg/core/inspection/taskbase` package.

---

## 2. Field Extraction with Extractor Functions

Logs in KHI provide a fast, zero-copy `*structured.NodeReader` via `l.NodeReader`.
To extract structured fields from raw logs, define an extractor function and a typed struct in the feature's root package:

```go
package myapp

import (
    "github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
    pathFoo = structured.CompileFieldPath("jsonPayload.foo")
    pathBar = structured.CompileFieldPath("jsonPayload.bar")
)

type MyFields struct {
    Foo string
    Bar int
}

func ExtractMyFields(reader *structured.NodeReader) (MyFields, error) {
    return MyFields{
        Foo: reader.ReadStringOrDefault(pathFoo, ""),
        Bar: reader.ReadIntOrDefault(pathBar, 0),
    }, nil
}
```

Tasks can then call `ExtractMyFields(l.NodeReader)` directly without intermediate task overhead or per-log hash table allocations.

---

## 3. Filtering Logs (`DefineLogFilterTask`)

Use `DefineLogFilterTask` to filter out noise logs (such as routine health check logs) in advance that are not needed for visualization or analysis.

```go
var MyFilterTask = inspectiontaskbase.DefineLogFilterTask(
    MyFilterTaskID,
    SourceLogsTaskID.Ref(),
    func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
        return func(ctx context.Context, l *log.Log) bool {
            fields, err := myapp.ExtractMyFields(l.NodeReader)
            return err == nil && fields.Bar > 0
        }
    },
)
```

---

## 4. Grouping Logs (`DefineLogGrouperTask`)

When an individual log message is not enough to identify a cause, you need to group related logs across time (e.g., a sequence of events from Pod creation to termination).
`DefineLogGrouperTask` groups logs based on a specific key:

```go
var MyGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
    MyGrouperTaskID,
    SourceLogsTaskID.Ref(),
    func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
        return func(ctx context.Context, l *log.Log) string {
            fields, err := myapp.ExtractMyFields(l.NodeReader)
            if err == nil && fields.Foo != "" {
                return fields.Foo
            }
            return "unknown"
        }
    },
)
```

---

## 5. Ingesting Logs (`DefineLogIngesterTask`)

`DefineLogIngesterTask` is an ingestion task (`coretask.Task[struct{}]`) that configures log-level metadata (such as log type, timestamp, severity, and summary) into a `*khifilev6.LogChangeSet`.
Pass the source log reference (`taskid.TaskReference[[]*log.Log]`) as the second argument and a `bind` function returning an `inspectiontaskbase.LogIngesterFunc` as the third argument:

```go
var MyLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
    MyLogIngesterTaskID,
    SourceLogsTaskID.Ref(),
    func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
        return func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
            cs, err := khifilev6.NewLogChangeSet(l)
            if err != nil {
                return nil, err
            }

            cs.SetTimestamp(l.Timestamp)
            cs.SetLogType(myapp.LogTypeMyApp)

            if fields, err := myapp.ExtractMyFields(l.NodeReader); err == nil {
                cs.SetSummary(fmt.Sprintf("[%s] count=%d", fields.Foo, fields.Bar))
            }

            return cs, nil
        }
    },
)
```

If your ingester needs additional upstream inputs, bind them on `b *coretask.Binder` inside the outer function and read their handles inside the returned `LogIngesterFunc`.

---

## 6. Mapping to Timelines (`DefineLogToTimelineMapperTask`)

As the final step in log processing, `DefineLogToTimelineMapperTask` maps `*log.Log` objects after filtering and grouping to resource trees and chronological timeline events (event bars, severity, detailed messages) rendered on the KHI UI.

### 6.1 Implementing the `TimelineMapper[T]` Interface

To create a mapper task, declare a struct that satisfies the `inspectiontaskbase.TimelineMapper[T]` interface.
In most cases, embed **`inspectiontaskbase.SinglePassMapperBase[T]`** (or `StatelessMapperBase`) to eliminate boilerplate and make it easy to write custom sequential processing, overriding only required methods.

```go
type MyGroupData struct {
    Count int
}

type MyMapper struct {
    inspectiontaskbase.SinglePassMapperBase[MyGroupData]
}

var _ inspectiontaskbase.TimelineMapper[MyGroupData] = (*MyMapper)(nil)
```

### 6.2 Creating and Using Timeline Path Helper Utilities

In current KHI implementations, mappers do not construct raw string paths directly. Instead, they use **`*khifilev6.TimelinePath`**, which clearly expresses resource hierarchies (tree structures) with types, to add and resolve events.
Furthermore, KHI uses a common implementation pattern where you create **timeline path helper utilities** (`MustXXXTimeline`) that compose parent timeline helper functions rather than rebuilding parent hierarchies with `TimelineAccumulator.GetPath` from scratch.

#### 1. Example of Creating a Timeline Path Helper

```go
// Example of a composite TimelinePath helper function in KHI composing parent helpers
func MustK8sPodTimeline(ctx context.Context, clusterName string, namespace string, podName string) *khifilev6.TimelinePath {
    clusterPath := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
    apiVersionPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
    kindPath := k8saudit.MustK8sKindTimeline(ctx, apiVersionPath, "pod")
    namespacePath := k8saudit.MustK8sNamespaceTimeline(ctx, kindPath, namespace)
    return k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespacePath, podName)
}
```

#### 2. Using the Timeline Path Helper from a Mapper

```go
// Process messages and add timeline events
func (m *MyMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevData MyGroupData) (*khifilev6.TimelineChangeSet, MyGroupData, error) {
    cs := khifilev6.NewTimelineChangeSet(l)

    // Call your timeline path helper utility to safely get the path
    podPath := MustK8sPodTimeline(ctx, "test-cluster", "default", "my-pod")

    // Add event
    cs.AddEvent(podPath)

    // Pass updated state to the next log processing step in the group
    return cs, MyGroupData{Count: prevData.Count + 1}, nil
}

// Initialize as a task
var MyMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
    MyMapperTaskID,
    inspectiontaskbase.TimelineMapperInputs{
        LogIngester: MyLogIngesterTaskID.Ref(),
        GroupedLogs: MyGrouperTaskID.Ref(),
    },
    func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[MyGroupData] {
        return &MyMapper{}
    },
    inspectioncore.FeatureTaskLabel(
        "Custom App Logs",
        "Parser and timeline mapping for Custom App logs.",
        9000,
        false,
    ),
)
```

### 6.3 Unit Testing Mappers (`testchangeset.AssertTimeline`)

When testing mappers, use `testchangeset.AssertTimeline(t, cs)` to declaratively verify the contents of the generated `TimelineChangeSet`.
Do not construct raw string paths; create and assert expected `*khifilev6.TimelinePath` objects.

If the mapper has no bound `coretask.Input[T]` handles, you can call `ProcessLogByGroup` directly. When the mapper reads bound inputs via `input.Get(ctx)`, extract the mapping logic into a pure helper function that accepts the resolved values directly (e.g., `mapMyLog(ctx, l, prevData, clusterIdentity)`), and test the task-level `Binder` wiring with `inspectiontest.Run(t, ctx, MyMapperTask, inspectioncore.TaskModeRun, nil, tasktest.Given(...))`.

```go
func TestMyMapper_ProcessLogByGroup(t *testing.T) {
    ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

    l := testlog.NewMockLog(time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC))
    mapper := &MyMapper{}

    cs, _, err := mapper.ProcessLogByGroup(ctx, l, MyGroupData{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    wantPodPath := MustK8sPodTimeline(ctx, "test-cluster", "default", "my-pod")
    testchangeset.AssertTimeline(t, cs).
        HasEvent(wantPodPath)
}
```

---

[< Previous: Syntax and Modes](./01-syntax-and-modes.md) | [Back to Index](../khi-task-system-concept.md) | [Next: Advanced Patterns >](./03-advanced-and-form-tasks.md)
