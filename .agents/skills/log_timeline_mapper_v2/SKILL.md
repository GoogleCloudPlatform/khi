---
name: log-timeline-mapper-v2
description: Guidelines for implementing and testing LogIngesterTaskV2 and LogToTimelineMapperV2 tasks in KHI.
---

# KHI Log Ingestion & Timeline Mapping Guidelines

This guide outlines the patterns, best practices, and testing methodologies for implementing log ingesters and timeline mappers in KHI.

---

## 1. LogIngesterV2 & LogIngesterTaskV2

`LogIngesterV2` is responsible for parsing raw logs and ingesting basic log metadata (summary, timestamp, severity, log type) into the KHI format.

### Interface Definition

```go
type LogIngesterV2 interface {
 // RawLogTask returns the task reference that provides the raw logs to ingest.
 RawLogTask() taskid.TaskReference[[]*log.Log]
 // Dependencies returns additional task dependencies of the ingester.
 Dependencies() []taskid.UntypedTaskReference
 // ProcessLog is called for each log entry to customize log metadata.
 ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error)
}
```

### Key Implementation Guide

> [!IMPORTANT]
> **ChangeSet Metadata Ingestion:** `LogChangeSet` does NOT automatically fill metadata defaults. You MUST explicitly read fields from `CommonFieldSet` (or custom field sets) and set them manually on the `LogChangeSet` in your `ProcessLog` implementation.
>
> **Skipping Logs:** If `ProcessLog` returns `(nil, nil)`, KHI will treat this log as skipped (ignored) without producing any errors.

#### Implementer Example

```go
type MyLogIngester struct {}

func (i *MyLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
 return rawLogTaskID.Ref()
}

func (i *MyLogIngester) Dependencies() []taskid.UntypedTaskReference {
 return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *MyLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
 // 1. Create a new change set.
 cs, err := khifilev6.NewLogChangeSet(l)
 if err != nil {
  return nil, err
 }

 // 2. Manually extract fields from CommonFieldSet.
 if commonSet, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
  cs.SetTimestamp(commonSet.Timestamp)
 }

 // 3. Set severity, summary, etc. manually using pre-registered styles.
 cs.SetSeverity(mySeverityStyle)
 cs.SetSummary(mySummaryString)

 return cs, nil
}

// Explicit interface compliance assertion is mandatory.
var _ inspectiontaskbase.LogIngesterV2 = (*MyLogIngester)(nil)
```

### Registering the LogIngester Task

> [!IMPORTANT]
> **Package Boundaries:**
>
> - **TaskID** definitions (e.g., `LogIngesterTaskIDV2`) MUST be defined in the `contract` package.
> - **Task Implementation** instantiations (e.g., `NewLogIngesterTaskV2`) MUST be placed in the `impl` package.

```go
// Defined in 'contract' package:
var MyLogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log]("my-log-ingester")

// Instantiated in 'impl' package:
task := NewLogIngesterTaskV2(mycontract.MyLogIngesterTaskID, &MyLogIngester{})
// Register task to core runner...
```

---

## 2. LogToTimelineMapperV2

`LogToTimelineMapperV2` maps grouped logs to timeline elements (events or resource revisions). Depending on the complexity, you should choose one of the following three implementation patterns.

### Pattern 1: Multi-Pass with State

Used for complex scenarios where you need to pre-collect information across all logs in a group before applying timeline changes (e.g., matching asynchronous request/response cycles).

- **How to implement:** Implement the full `LogToTimelineMapperV2[T]` interface manually.

```go
type ComplexMapper struct {}

func (m *ComplexMapper) PassCount() int {
 return 1 // Run 1 pre-processing pass.
}

func (m *ComplexMapper) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData MyState) (MyState, error) {
 // Pre-collect state from logs.
 nextState := analyzeLog(prevGroupData, l)
 return nextState, nil
}

func (m *ComplexMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData MyState) (*khifilev6.TimelineChangeSet, MyState, error) {
 // 1. Retrieve field data using log.GetFieldSet.
 customSet, err := log.GetFieldSet(l, &MyCustomFieldSet{})
 if err != nil {
  return nil, prevGroupData, err
 }

 // 2. Retrieve the Builder from context.
 builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

 // 3. Resolve target path dynamically using the accumulator facade.
 targetPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: "complex-timeline",
  Type: mycontract.TimelineTypeComplex, // Timeline styles should be imported from contract package
 })

 cs := khifilev6.NewTimelineChangeSet(l)

 // Add a revision or event conditionally using the pre-collected state and customSet fields.
 if prevGroupData.ShouldRegisterRevision(l) {
  cs.AddRevision(targetPath, &khifilev6.StagingRevision{
   ChangedTime:  customSet.Timestamp,
   ResourceBody: customSet.Body,
   Principal:    customSet.Principal,
   VerbType:     mycontract.VerbCreate,
  })
 }

 return cs, prevGroupData, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogToTimelineMapperV2[MyState] = (*ComplexMapper)(nil)
```

---

### Pattern 2: Single-Pass with State

Used when you need to maintain and propagate state sequentially through the logs in a group, but do not require a pre-processing pass.

- **How to implement:** Embed `SinglePassMapperBase[T]` into your mapper structure. This automatically implements `PassCount() int` (returning 0) and `PreProcessLogByGroup` (returning state as-is).

```go
type StateTrackingMapper struct {
 SinglePassMapperBase[MyState] // Embeds single pass helper.
}

func (m *StateTrackingMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData MyState) (*khifilev6.TimelineChangeSet, MyState, error) {
 // 1. Retrieve field data using log.GetFieldSet.
 customSet, err := log.GetFieldSet(l, &MyCustomFieldSet{})
 if err != nil {
  return nil, prevGroupData, err
 }

 // 2. Maintain state.
 nextState := updateState(prevGroupData, customSet)

 // 3. Retrieve the Builder from context.
 builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

 // 4. Resolve target path dynamically using the accumulator facade.
 targetPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: "stateful-revision-timeline",
  Type: mycontract.TimelineTypeStateful,
 })

 cs := khifilev6.NewTimelineChangeSet(l)

 // Append resource revision history sequentially to the timeline.
 cs.AddRevision(targetPath, &khifilev6.StagingRevision{
  ChangedTime:  customSet.Timestamp,
  ResourceBody: customSet.Body,
  Principal:    customSet.Principal,
  VerbType:     mycontract.VerbUpdate,
 })

 return cs, nextState, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogToTimelineMapperV2[MyState] = (*StateTrackingMapper)(nil)
```

---

### Pattern 3: Single-Pass Stateless (Most Common)

Used when timeline mapping for each log is completely independent and does not rely on other logs in the same group.

- **How to implement:** Embed `StatelessMapperBase` into your mapper structure. This binds the state type `T` to `struct{}` and implements the pre-processing methods as no-ops.

```go
type SimpleEventMapper struct {
 StatelessMapperBase // Embeds stateless helper.
}

func (m *SimpleEventMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
 // 1. Retrieve the Builder from context.
 builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

 // 2. Resolve target path dynamically.
 targetPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: "simple-event-timeline",
  Type: mycontract.TimelineTypeEvent,
 })

 cs := khifilev6.NewTimelineChangeSet(l)

 // Add a simple timeline event.
 cs.AddEvent(targetPath)

 return cs, struct{}{}, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*SimpleEventMapper)(nil)
```

### Registering the TimelineMapper Task

> [!IMPORTANT]
> **Package Boundaries:**
>
> - **TaskID** definitions (e.g., `LogToTimelineMapperTaskIDV2`) MUST be defined in the `contract` package.
> - **Task Implementation** instantiations (e.g., `NewLogToTimelineMapperTaskV2`) MUST be placed in the `impl` package.

```go
// Defined in 'contract' package:
var MyTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}]("my-timeline-mapper")

// Instantiated in 'impl' package:
task := NewLogToTimelineMapperTaskV2(mycontract.MyTimelineMapperTaskID, &SimpleEventMapper{})
// Register task to core runner...
```

---

## 3. Testing Guidelines

Tests for V2 tasks must follow the standard Table-Driven testing pattern. To verify mappers or ingesters produced correct outcomes across various scenarios, you should write unit tests utilizing `testchangeset` fluent assertions.

### Table-Driven Fluent Assertions

KHI provides a dedicated test utility `github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset` to perform readable assertions against staged changesets. By incorporating `testchangeset.AssertLog` or `testchangeset.AssertTimeline` into your table-driven loop, you can verify multiple test cases cleanly and expressively.

#### LogIngester Table-Driven Assertion Example

To isolate ingester parsing logic, instantiate logs using `log.NewLogWithFieldSetsForTest` and define chainable assertions inside test cases:

```go
func TestMyLogIngester_ProcessLog(t *testing.T) {
 testCases := []struct {
  name   string
  input  *log.Log
  assert func(t *testing.T, cs *khifilev6.LogChangeSet)
 }{
  {
   name: "successful info log ingestion",
   input: log.NewLogWithFieldSetsForTest(
    &log.CommonFieldSet{
     Timestamp: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
     Severity:  "INFO",
    },
    &googlecloudcommon_contract.GCPMainMessageFieldSet{
     MainMessage: "server started",
    },
   ),
   assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
    testchangeset.AssertLog(t, cs).
     HasSummary("server started").
     HasSeverity(infoSeverityStyle)
   },
  },
 }

 ingester := &MyLogIngester{}
 for _, tc := range testCases {
  t.Run(tc.name, func(t *testing.T) {
   // Obtain context dynamically using t.Context()
   ctx := t.Context()

   cs, err := ingester.ProcessLog(ctx, tc.input)
   if err != nil {
    t.Fatalf("ProcessLog() returned unexpected error: %v", err)
   }

   tc.assert(t, cs)
  })
 }
}
```

#### TimelineMapper Table-Driven Assertion Example

Mappers translate structured log `FieldSet` data into timeline changes. Isolate mapper tests using `log.NewLogWithFieldSetsForTest` and execute assertions using the fluent `changeset` asserter.

> [!IMPORTANT]
> **Shared Builder Reference:** When unit testing mappers that dynamically construct timeline paths via context builder, you MUST initialize a single `khifilev6.Builder` and resolve all comparison `TimelinePath` instances using this builder. Crucially, the same builder instance must be injected into the execution context using `khictx.WithValue` to ensure pointer equality during assertions.

```go
func TestMyTimelineMapper_ProcessLogByGroup(t *testing.T) {
 // 1. Initialize the Builder first.
 builder := khifilev6.NewBuilder()

 // 2. Resolve comparative path instances using the Builder's accumulator.
 // TimelineTypes must be imported from the contract package.
 resourceTimelinePath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: "resource-timeline",
  Type: mycontract.TimelineTypeResource,
 })

 testCases := []struct {
  name      string
  inputLog  *log.Log
  prevState MyState
  assert    func(t *testing.T, cs *khifilev6.TimelineChangeSet)
 }{
  {
   name: "create resource revision",
   inputLog: log.NewLogWithFieldSetsForTest(&MyCustomFieldSet{
    Verb: "create",
   }),
   prevState:      MyState{},
   assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
    testchangeset.AssertTimeline(t, cs).
     HasEvent(resourceTimelinePath).
     HasRevision(resourceTimelinePath, &khifilev6.StagingRevision{
      VerbType: mycontract.VerbCreate,
     })
   },
  },
  {
   name: "skip timeline revision on delete verb",
   inputLog: log.NewLogWithFieldSetsForTest(&MyCustomFieldSet{
    Verb: "delete",
   }),
   prevState:      MyState{},
   assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
    testchangeset.AssertTimeline(t, cs).
     HasNoEvent(resourceTimelinePath).
     HasNoRevision(resourceTimelinePath)
   },
  },
 }

 mapper := &MySimpleMapper{}
 for _, tc := range testCases {
  t.Run(tc.name, func(t *testing.T) {
   // 3. Set up the context using t.Context() and SAME builder instance.
   ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

   cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, tc.prevState)
   if err != nil {
    t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
   }

   tc.assert(t, cs)
  })
 }
}
```
