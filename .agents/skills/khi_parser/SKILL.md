---
name: khi-parser
description: Guidelines, package patterns, and task implementations for adding new log type support or modifying existing log parsers in KHI.
---

# KHI Log Parser Support Guidelines

This guide outlines the patterns, package boundaries, implementation steps, and best practices for adding support for new log types or modifying existing log parsers in KHI.

---

## 1. Package Structure & Boundaries

When implementing a new log parser or modifying an existing one, you MUST separate the **contract** (IDs, public types, and configurations) from the **implementation** (the actual task logic). This guarantees that task IDs are fully initialized before implementation and prevents circular import dependencies.

The parser package must reside under `pkg/task/inspection/` and adhere to the following structure:

```plaintext
pkg/task/inspection/<log_type_name>/
├── contract/
│   ├── taskid.go          // Defines all TaskIDs and TaskReferences.
│   ├── fieldset.go        // (Optional) Defines the strongly-typed FieldSets and their Readers.
│   ├── timeline_type.go   // (Optional) Defines timeline types and verb types.
│   ├── timeline_path.go   // (Optional) Helper functions to build hierarchical paths.
│   └── log_type.go        // (Optional) Defines log-specific types or constants.
└── impl/
    ├── form_task.go       // (Optional) Implements form-related parameter tasks.
    ├── query_task.go      // (Optional) Implements log query/filter tasks.
    ├── ingester_task.go   // (Optional) Implements the LogIngester task.
    ├── <name>_mapper.go   // (Optional) Implements LogToTimelineMapper tasks (can be multiple).
    └── registration.go    // Implements task registration to the KHI registry.
```

### Key Package Boundaries

> [!IMPORTANT]
>
> - **Contract Package (`contract/`)**: MUST NOT import the `impl` package. External packages can freely import the `contract` package to depend on parser task IDs, FieldSet types, or TimelineType constants.
> - **Implementation Package (`impl/`)**: Implements the actual tasks. It imports the `contract` package. **External packages MUST NOT import the `impl` package.**
> - **Registration**: Tasks inside the `impl` package are registered through `impl/registration.go`. There is no root-level `registration.go` file in this directory.

---

## 2. The Five Log Parsing Steps

A complete log parser in KHI generally consists of 5 sequential steps, represented as distinct DAG tasks:

```mermaid
flowchart TD
    FormTask[1. Form Task] -->|Provides Parameters| QueryTask[2. Log Query Task]
    QueryTask -->|Provides Raw Logs| FieldSetTask[3. FieldSet Reading Task]
    FieldSetTask -->|Provides Parsed FieldSets| IngesterTask[4. Log Ingest Task]
    FieldSetTask -->|Provides Parsed FieldSets| GrouperTask[Log Grouper Task]
    IngesterTask -->|Provides Ingested Logs| MapperTask[5. Timeline Mapper Task]
    GrouperTask -->|Provides Grouped Map| MapperTask
```

### Step 1: Form Tasks (Form-related)

Exposes interactive input fields (e.g., text boxes, multi-select checkboxes) to let users configure parameters before running the inspection.

- **Utility:** `formtask.NewTextFormTaskBuilder` or `formtask.NewSetFormTaskBuilder`.

### Step 2: Log Query Tasks

Queries logs from the data source (e.g., Google Cloud Logging or local files) using parameters provided by the Form tasks.

- **Utility:** `googlecloudcommon_contract.NewListLogEntriesTask` (for any logs on Cloud Logging) or `inspection_task.NewInspectionTask`.

### Step 3: FieldSet Reading Tasks

Reads structured log fields in parallel and attaches strongly-typed structs called **FieldSets** to log objects so subsequent tasks can access them efficiently.

- **Utility:** `inspectiontaskbase.NewFieldSetReadTask` with custom `log.FieldSetReader`s.

### Step 4: Log Ingestion Tasks

Information linked to the log is parsed here into a writable format, receiving information from the FieldSets (e.g. Timestamp, Severity, and Summary are initialized and stage mutations).

- **Utility:** `inspectiontaskbase.NewLogIngesterTaskV2`.

### Step 5: Timeline Mapping Tasks

Maps the ingested logs to resource timelines as events or state revisions.

- **Utility:** `inspectiontaskbase.NewLogToTimelineMapperTaskV2`.

---

## 3. Step-by-Step Implementation Code Samples

Let's look at a concrete example of supporting a custom log type called `customapp`.

### A. The Contract Package (`pkg/task/inspection/customapp/contract/`)

#### `taskid.go`

Defines the TaskIDs and TaskReferences for all 5 steps.

```go
package customapp_contract

import (
 inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
 "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
 "github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const TaskIDPrefix = "customapp.khi.google.com/"

// 1. Form Task ID
var InputFilterKeywordTaskID = taskid.NewDefaultImplementationID[string](TaskIDPrefix + "input-keyword")

// 2. Log Query Task ID
var LogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// 3. FieldSet Reading Task ID
var FieldSetReadTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "fieldset-read")

// 4. Log Ingestion Task ID
var LogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-ingester")

// 5. Log Grouper & Timeline Mapper Task IDs
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "log-grouper")
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "timeline-mapper")
```

#### `fieldset.go`

Defines the strongly-typed FieldSet and its reader.

```go
package customapp_contract

import (
 "github.com/GoogleCloudPlatform/khi/pkg/common/structured"
 "github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// CustomAppFieldSet holds structured log data extracted from the log entry.
type CustomAppFieldSet struct {
 AppName   string
 RequestID string
 Payload   string
}

// Kind returns the unique identifier of this FieldSet.
func (fs *CustomAppFieldSet) Kind() string {
 return "customapp.khi.google.com/FieldSet"
}

// CustomAppFieldSetReader extracts CustomAppFieldSet from a raw log node reader.
type CustomAppFieldSetReader struct{}

// FieldSetKind returns the Kind string of CustomAppFieldSet.
func (r *CustomAppFieldSetReader) FieldSetKind() string {
 return "customapp.khi.google.com/FieldSet"
}

// Read parses fields from structured log data.
func (r *CustomAppFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
 return &CustomAppFieldSet{
  AppName:   reader.ReadStringOrDefault("app_name", "unknown-app"),
  RequestID: reader.ReadStringOrDefault("request_id", ""),
  Payload:   reader.ReadStringOrDefault("payload", ""),
 }, nil
}

var _ log.FieldSetReader = (*CustomAppFieldSetReader)(nil)
```

#### `timeline_type.go`

Defines custom timeline types and resource verbs.

```go
package customapp_contract

import (
 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
 // TimelineTypeCustomApp is the timeline type style for Custom App resources.
 TimelineTypeCustomApp = style.MustRegisterTimelineType(
  "customapp",
  "Custom Application",
  "dns",
  0.6,
  style.ColorWhite,
  style.ColorWhite,
  style.MustForceConvertSRGBHex("#4285F4"),
  true,
  1000,
 )

 // VerbCustomAppProcess is the verb style for Custom App state updates.
 VerbCustomAppProcess = style.MustRegisterVerb("Process", style.MustForceConvertSRGBHex("#0F9D58"), style.ColorWhite, true)
)
```

#### `log_type.go`

Defines custom log types.

```go
package customapp_contract

import (
 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

var (
 // LogTypeCustomApp is the log type style for Custom App logs.
 LogTypeCustomApp = style.MustRegisterLogType(
  "customapp",
  "Custom Application Logs",
  style.MustForceConvertSRGBHex("#4285F4"),
  style.ColorWhite,
 )
)
```

#### `timeline_path.go` (Optional)

Defines helper functions to build hierarchical timeline paths.

For custom application timelines, you can define helpers to construct paths consistently. If your custom application runs as part of a Kubernetes Pod, you can build a sub-timeline path nested directly under the standard Kubernetes Pod timeline by referencing standard K8s timeline types from `inspectioncore_contract`.

- MustXXXTimeline func must receive the context as its first argument.
- If the MustXXXTimeline func isn't for a root timeline, it must receive the parent timeline path as its second argument.

```go
package customapp_contract

import (
 "context"

 "github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
 khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
 inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// MustCustomAppTimeline returns the hierarchical timeline path for a standalone Custom App.
// Constructs a path like: customapp/<appName>
func MustCustomAppTimeline(ctx context.Context, appName string) *khifilev6.TimelinePath {
 builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
 return builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: appName,
  Type: TimelineTypeCustomApp,
 })
}

// MustCustomAppPodTimeline returns the hierarchical timeline path for Custom App logs nested under a Pod.
// Constructs a path like: <apiVersion>/<kind>/<namespace>/<podName>/customapp
func MustCustomAppPodTimeline(ctx context.Context, podTimelinePath *khifilev6.TimelinePath) *khifilev6.TimelinePath {
  if podTimelinePath == nil || podTimelinePath.Type.GetId() != inspectioncore_contract.TimelineTypeResource.GetId() {
  panic("parent timeline path must be Resource type")
 }

 builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
 return builder.TimelineAccumulator.GetPath(podTimelinePath, khifilev6.PathSegment{
  Name: "customapp",
  Type: TimelineTypeCustomApp,
 })
}
```

---

### B. The Implementation Package (`pkg/task/inspection/customapp/impl/`)

#### `form_task.go` (Step 1)

Implements form tasks to get user-defined input.

```go
package customapp_impl

import (
 "context"

 "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
 customapp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/customapp/contract"
 googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

const formPriority = googlecloudcommon_contract.FormBasePriority + 5000

// InputFilterKeywordTask defines a text input form task for filtering logs.
var InputFilterKeywordTask = formtask.NewTextFormTaskBuilder(
 customapp_contract.InputFilterKeywordTaskID,
 formPriority,
 "Filter Keyword",
).
 WithDescription("Keyword to filter Custom App logs.").
 WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
  if len(previousValues) > 0 {
   return previousValues[0], nil
  }
  return "default-keyword", nil
 }).
 Build()
```

#### `query_task.go` (Step 2)

Implements querying logs from Google Cloud Logging based on parameters.

```go
package customapp_impl

import (
 "context"
 "fmt"

 coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
 "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"

 "github.com/GoogleCloudPlatform/khi/pkg/model/log"
 googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
 googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
 customapp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/customapp/contract"
 inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// LogQueryTask executes Cloud Logging filter to fetch logs.
var LogQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&customAppLogQueryTaskSetting{})

type customAppLogQueryTaskSetting struct{}

func (s *customAppLogQueryTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
 return customapp_contract.LogQueryTaskID
}

func (s *customAppLogQueryTaskSetting) Dependencies() []taskid.UntypedTaskReference {
 return []taskid.UntypedTaskReference{
  googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
  customapp_contract.InputFilterKeywordTaskID.Ref(),
 }
}

func (s *customAppLogQueryTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
 return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
  DefaultLogType: customapp_contract.LogTypeCustomApp,
  QueryName:      "Custom App logs",
  ExampleQuery:   `resource.type="gke_cluster" AND log_id("custom-app")`,
 }
}

func (s *customAppLogQueryTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
 keyword := coretask.GetTaskResult(ctx, customapp_contract.InputFilterKeywordTaskID.Ref())
 query := fmt.Sprintf(`resource.type="gke_cluster" AND log_id("custom-app") AND textPayload:"%s"`, keyword)
 return []string{query}, nil
}

func (s *customAppLogQueryTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
 clusterIdentity := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
 return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

func (s *customAppLogQueryTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
 return 5, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*customAppLogQueryTaskSetting)(nil)
```

#### `parser_tasks.go` (Steps 3, 4, 5)

Defines FieldSet reading, log ingestion, log grouping, and timeline mapping.

```go
package customapp_impl

import (
 "context"
 "fmt"

 "github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
 inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
 "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
 khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"

 "github.com/GoogleCloudPlatform/khi/pkg/model/log"
 customapp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/customapp/contract"
 googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
 inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReadTask reads fields in parallel (Step 3).
var FieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
 customapp_contract.FieldSetReadTaskID,
 customapp_contract.LogQueryTaskID.Ref(),
 []log.FieldSetReader{
  &customapp_contract.CustomAppFieldSetReader{},
  &googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
 },
)

// CustomAppLogIngester V2 LogIngester (Step 4).
type CustomAppLogIngester struct{}

func (i *CustomAppLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
 return customapp_contract.FieldSetReadTaskID.Ref()
}

func (i *CustomAppLogIngester) Dependencies() []taskid.UntypedTaskReference {
 return []taskid.UntypedTaskReference{}
}

func (i *CustomAppLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
 cs, err := khifilev6.NewLogChangeSet(l)
 if err != nil {
  return nil, err
 }
  // Log related parsing logic. LogType, Timestamp, Severity and Summary must be set.

 cs.SetLogType(foocontract.LogTypeFoo)
 // Explicitly retrieve and ingest Timestamp from CommonFieldSet.
 if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
  cs.SetTimestamp(commonFS.Timestamp)
  // Do not use commonFS.Severity because it's deprecated.
 }

 // Retrieve and ingest Severity from DefaultSeverityFieldSet.
 if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
  cs.SetSeverity(severityFS.Severity)
 }

 // Retrieve custom fields to generate summary.
 if customFS, err := log.GetFieldSet(l, &customapp_contract.CustomAppFieldSet{}); err == nil {
  cs.SetSummary(fmt.Sprintf("[%s] %s", customFS.AppName, customFS.Payload))
 }

 return cs, nil
}

var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
 customapp_contract.LogIngesterTaskID,
 &CustomAppLogIngester{},
)

// LogGrouperTask groups logs byAppName (helper for Step 5).
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
 customapp_contract.LogGrouperTaskID,
 customapp_contract.FieldSetReadTaskID.Ref(),
 func(ctx context.Context, l *log.Log) string {
  if customFS, err := log.GetFieldSet(l, &customapp_contract.CustomAppFieldSet{}); err == nil {
   return customFS.AppName
  }
  return "unknown-app"
 },
)

// CustomAppTimelineMapper maps logs to timeline (Step 5).
type CustomAppTimelineMapper struct {
 inspectiontaskbase.StatelessMapperBase // Embed stateless helper.
}

func (m *CustomAppTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
 return customapp_contract.LogIngesterTaskID.Ref()
}

func (m *CustomAppTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
 return []taskid.UntypedTaskReference{}
}

func (m *CustomAppTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
 return customapp_contract.LogGrouperTaskID.Ref()
}

func (m *CustomAppTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
 commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{})
 if err != nil {
  return nil, struct{}{}, err
 }
 customFS, err := log.GetFieldSet(l, &customapp_contract.CustomAppFieldSet{})
 if err != nil {
  return nil, struct{}{}, err
 }

 builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentV6Builder)
 targetPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
  Name: customFS.AppName,
  Type: customapp_contract.TimelineTypeCustomApp,
 })

 cs := khifilev6.NewTimelineChangeSet(l)

 // Record a revision on timeline for state change.
 cs.AddRevision(targetPath, &khifilev6.StagingRevision{
  ChangedTime:  commonFS.Timestamp,
  ResourceBody: customFS.Payload,
  VerbType:     customapp_contract.VerbCustomAppProcess,
 })

 return cs, struct{}{}, nil
}

var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
 customapp_contract.LogToTimelineMapperTaskID,
 &CustomAppTimelineMapper{},
 inspectioncore_contract.FeatureTaskLabelV2(
  "Custom App Logs",
  "Parser and timeline mapping for Custom App logs.",
  9000,
  false,
 ),
)

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*CustomAppTimelineMapper)(nil)
```

#### `registration.go`

Registers the tasks with the central registry.

```go
package customapp_impl

import (
 coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
 coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// Register registers all customapp tasks to the central registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
 return coretask.RegisterTasks(
  registry,
  InputFilterKeywordTask,
  LogQueryTask,
  FieldSetReadTask,
  LogIngesterTask,
  LogGrouperTask,
  LogToTimelineMapperTask,
 )
}
```

---

## 4. Testing Log Parsers

Tests must follow the standard Table-Driven pattern and make use of fluent `testchangeset` assertion helpers for verification.

Refer to [log-timeline-mapper-v2](skill://log-timeline-mapper-v2) for detailed unit testing strategies of `LogIngesterV2` and `LogToTimelineMapperV2`.
