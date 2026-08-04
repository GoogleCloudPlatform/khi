# Log Processing Task Implementation Patterns (Cookbook)

[< Previous: Syntax and Modes](./01-syntax-and-modes.md) | [Back to Index](../khi-task-system-concept.md) | [Next: Advanced Patterns >](./03-advanced-and-form-tasks.md)

---

This document explains the overall log processing pipeline in KHI and provides **practical cookbooks for 5 high-level task utilities** that developers use to implement new log parsers and timeline mappers.

## 1. Overview of the Log Processing Pipeline

KHI uses its robust task system to perform log processing. This architecture makes KHI highly extensible (you can add new features just by creating new tasks) and allows it to fully leverage Go's concurrency.
However, because log processing shares many common patterns, you should implement log processing tasks using the high-level task creation utilities provided by KHI.

KHI provides the following high-level task creation utilities to cover basic log processing use cases:

- **`FieldSetReadTask`** : Stores structured log fields into typed structs.
- **`LogGrouperTask`** : Groups logs by specific fields.
- **`LogFilterTask`** : Filters logs based on conditions.
- **`LogIngesterTask`** : Ingests logs into the final history data.
- **`LogToTimelineMapperTask`** : Maps ingested logs to timeline events displayed on the KHI UI.

The following is an example dependency graph showing how these 5 task types work together to parse Kubernetes node audit logs fetched from Cloud Logging and render them on the UI timeline:

```mermaid
flowchart TD
    Fetch["Log Collection/Query Task<br>(Cloud Logging, etc.)"]
    Ingester["LogIngesterTask<br>(Build log objects and inject types)"]
    FieldSet["FieldSetReadTask<br>(Read specific sets of fields)"]
    Filter["LogFilterTask<br>(Filter out unwanted logs)"]
    Grouper["LogGrouperTask<br>(Group by Pod name or thread)"]
    Mapper["LogToTimelineMapperTask<br>(Map to timeline events and paths)"]

    Fetch --> Ingester
    Ingester --> FieldSet
    FieldSet --> Filter
    Filter --> Grouper
    Grouper --> Mapper
```

When developers add support for a new log type, they declare tasks along this pipeline and connect them to the task graph.
You create all of these using utility functions in the `pkg/core/inspection/taskbase` package (such as `NewFieldSetReadTask`).

---

## 2. Reading Field Sets (`FieldSetReadTask`)

Logs in KHI are unstructured initially, so you should use `FieldSetReadTask` to read specific fields from logs.
This unmarshals and stores a specific set of fields from a log into a predefined Go struct type.

```go
// 1. Define a struct for the fields you want to read
type MyFieldSet struct {
    foo string
    bar int
}

// 2. Define a task that attaches the field set to log structures
var MyFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
    MyFieldSetReadTaskID, // The ID of this task itself
    SourceLogsTaskID.Ref(), // The target log list
    func(ctx context.Context, l *log.Log) (*MyFieldSet, error) {
        // ... (Logic to extract values from the log body) ...
        return &MyFieldSet{
            foo: "foo",
            bar: 1,
        }, nil
    },
)
```

In tasks that use this utility, you can read the specific field set from a log using `log.GetFieldSet(l, &MyFieldSet{})`.

```go
fieldSet := log.GetFieldSet(l, &MyFieldSet{})
```

> [!TIP]
> For details on why we attach information to log objects instead of returning values directly from tasks, see **"Log Immutability and Concurrent Access"** in the architecture documentation.

---

## 3. Grouping Logs (`LogGrouperTask`)

When an individual log message is not enough to identify a cause, you need to group related logs across time (e.g., a sequence of events from Pod creation to termination).
`LogGrouperTask` groups logs based on a specific key:

```go
var MyGrouperTask = inspectiontaskbase.NewLogGrouperTask(
    MyGrouperTaskID,
    SourceLogsTaskID.Ref(),
    func(ctx context.Context, l *log.Log) (string, error) {
        // Return a string as the grouping key from log l
        fieldSet := log.GetFieldSet(l, &MyFieldSet{})
        return fieldSet.foo, nil
    },
)
```

Subsequent tasks can use `log.GetGroup(l, MyGrouperTaskID)` to get the group name (group key) that a target log belongs to.

---

## 4. Filtering Logs (`LogFilterTask`)

Use `LogFilterTask` to filter out noise logs (such as routine health check logs) in advance that are not needed for visualization or analysis.

```go
var MyFilterTask = inspectiontaskbase.NewLogFilterTask(
    MyFilterTaskID,
    SourceLogsTaskID.Ref(),
    func(ctx context.Context, l *log.Log) (bool, error) {
        // Keep logs that return true, and filter out logs that return false
        fieldSet := log.GetFieldSet(l, &MyFieldSet{})
        return fieldSet.bar > 0, nil
    },
)
```

---

## 5. Ingesting Logs (`LogIngesterTask`)

`LogIngesterTask` is an ingestion task that initializes and appends raw log strings collected from cloud APIs or local files into common `*log.Log` objects in KHI, and executes parser initialization logic for each log type (such as generating summaries and injecting required field sets).

### 1. Declaring the `LogIngester` Interface

When defining an ingestion task, first implement a struct that satisfies the `inspectiontaskbase.LogIngester` interface:

```go
type MyLogIngester struct{}

// RawLogTask returns the raw log source or parent task reference ID
func (i *MyLogIngester) RawLogTask() taskid.UntypedTaskReference {
    return SourceLogsTaskID.Ref()
}

// Dependencies returns the list of dependencies required by the parser
func (i *MyLogIngester) Dependencies() []taskid.UntypedTaskReference {
    return []taskid.UntypedTaskReference{}
}

// ProcessLog implements initial processing or transformation logic for each log
func (i *MyLogIngester) ProcessLog(ctx context.Context, l *log.Log) error {
    // Summarize log messages and initialize basic fields
    l.Summary = "Parsed Log Summary"
    return nil
}

var _ inspectiontaskbase.LogIngester = (*MyLogIngester)(nil)
```

### 2. Building the Task Instance

Pass the address of your ingester struct to `inspectiontaskbase.NewLogIngesterTask` to declare the task instance:

```go
var MyLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
    MyLogIngesterTaskID,
    &MyLogIngester{},
)
```

---

## 6. Mapping to Timelines (`LogToTimelineMapperTask`)

As the final step in log processing, `LogToTimelineMapperTask` maps `*log.Log` objects after filtering and grouping to resource trees and chronological timeline events (event bars, severity, detailed messages) rendered on the KHI UI.

### 6.1 Implementing the `LogToTimelineMapper[T]` Interface

To create a mapper task, declare a struct that satisfies the `inspectiontaskbase.LogToTimelineMapper[T]` interface.
In most cases, embed **`inspectiontaskbase.SinglePassMapperBase[T]`** to eliminate boilerplate and make it easy to write custom sequential processing, overriding only required methods.

```go
type MyGroupData struct {
    Count int
}

type MyMapper struct {
    inspectiontaskbase.SinglePassMapperBase[MyGroupData]
}

func (m *MyMapper) GroupingTask() taskid.UntypedTaskReference {
    return MyGrouperTaskID.Ref()
}

func (m *MyMapper) LogTask() taskid.UntypedTaskReference {
    return SourceLogsTaskID.Ref()
}

func (m *MyMapper) Dependencies() []taskid.UntypedTaskReference {
    return []taskid.UntypedTaskReference{MyFieldSetReadTaskID.Ref()}
}
```

### 6.2 Creating and Using Timeline Path Helper Utilities

In current KHI implementations, mappers do not construct raw string paths directly. Instead, they use **`*khifilev6.TimelinePath`**, which clearly expresses resource hierarchies (tree structures) with types, to add and resolve events.
Furthermore, KHI uses a common implementation pattern where you create a **timeline path helper utility** that encapsulates joining child segments to parent paths, and reuses objects through `TimelinePathPool`.

#### 1. Example of Creating a Timeline Path Helper (defined in `contract` package, etc.)

```go
// Example of a standard TimelinePath helper function in KHI
func MustK8sPodTimeline(ctx context.Context, clusterName string, namespace string, podName string) *khifilev6.TimelinePath {
    // 1. Get or construct the parent cluster TimelinePath
    clusterPath := MustK8sClusterTimeline(ctx, clusterName)
    // 2. Use TimelinePathPool to safely join child segments and avoid duplicate allocations
    pathPool := khictx.MustGetValue(ctx, inspectioncore_contract.TimelinePathPool)
    return pathPool.MustGet(clusterPath, namespace, podName)
}
```

#### 2. Using the Timeline Path Helper from a Mapper

```go
// Process messages and add timeline events
func (m *MyMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevData MyGroupData) (*khifilev6.TimelineChangeSet, MyGroupData, error) {
    cs := khifilev6.NewTimelineChangeSet()

    // Call your timeline path helper utility to safely get the path
    podPath := commonlogk8saudit_contract.MustK8sPodTimeline(ctx, "test-cluster", "default", "my-pod")

    // Add event
    cs.AddEvent(podPath)

    // Pass updated state to the next log processing step in the group
    return cs, MyGroupData{Count: prevData.Count + 1}, nil
}

// Initialize as a task
var MyMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
    MyMapperTaskID,
    &MyMapper{},
    inspectioncore_contract.FeatureTaskLabel("my-feature", "Feature label", "Detailed description of the feature", true, "gcp-gke"),
)
```

### 6.3 Unit Testing Mappers (`testchangeset.AssertTimeline`)

When testing mappers, use `testchangeset.AssertTimeline(t, cs)` to declaratively verify the contents of the generated `TimelineChangeSet`.
Do not use string paths or deprecated APIs; create and assert expected `*khifilev6.TimelinePath` objects.

```go
func TestMyMapper_ProcessLogByGroup(t *testing.T) {
    l := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{Timestamp: time.Now()}, /* ... */)
    mapper := &MyMapper{}

    cs, _, err := mapper.ProcessLogByGroup(t.Context(), l, MyGroupData{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    wantPodPath := commonlogk8saudit_contract.MustK8sPodTimeline(t.Context(), "test-cluster", "default", "my-pod")
    testchangeset.AssertTimeline(t, cs).
        HasEvent(wantPodPath).
        HasLogSeverity(enum.SeverityInfo)
}
```

---

[< Previous: Syntax and Modes](./01-syntax-and-modes.md) | [Back to Index](../khi-task-system-concept.md) | [Next: Advanced Patterns >](./03-advanced-and-form-tasks.md)
