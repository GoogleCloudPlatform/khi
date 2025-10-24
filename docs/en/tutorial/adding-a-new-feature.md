# Tutorial: Adding a BigQuery Job Visualizer

This tutorial guides you through the process of adding a new feature to the Kubernetes History Inspector (KHI). We will use the implementation of the BigQuery Job Visualizer as a practical, real-world example.

By the end of this tutorial, you will understand how to:

- Design and plan a new feature for KHI.
- Add new log types, states, and verbs to the data model.
- Implement a custom parser for a new data source using KHI's task system.
- Write unit tests for your new feature.
- Register your feature to make it available in the KHI application.
- Understand how the frontend consumes the data you've added.

## Part 1: Prerequisites

Before you begin, please ensure you have:

1. A working KHI development environment. If you haven't set one up, please follow the instructions in the [README.md](../../../README.md#run-from-source-code).
2. A basic understanding of the Go programming language.
3. Read the [KHI Task system concept](../khi-task-system-concept.md) document. This is essential for understanding KHI's core architecture.

## Part 2: The Big Picture: How KHI Features Work

KHI's power comes from its modular, task-based architecture. Adding a new feature, like our BigQuery visualizer, involves creating a series of tasks that form a small part of a larger Directed Acyclic Graph (DAG).

Here's the overall flow for our new feature:

1. **User Input**: The user selects the "BigQuery" inspection type in the UI.
2. **Query Generation**: A task generates a specific query to fetch BigQuery `jobCompletedEvent` logs from Google Cloud Logging.
3. **Log Fetching**: KHI's core system uses this query to retrieve the logs.
4. **Parsing**: Our new parser task receives each log entry.
5. **History Building**: The parser extracts key information (like job creation, start, and end times) and transforms it into "revisions" in KHI's internal data model.
6. **Frontend Rendering**: The frontend reads this structured history data and visualizes it as a timeline.

Our goal in this tutorial is to implement the tasks and data model changes required for this flow.

## Part 3: Defining Core Types and Enums

The first step in implementing our feature is to extend KHI's core data model to understand the concepts related to BigQuery jobs. This is done in the `pkg/model/enum/` package.

These enums are primarily for the frontend. In KHI, the frontend contains very little information itself; instead, it renders dynamically based on the data sent from the backend.

### Step 3.1: Add a new Log Type

We need a new `LogType` to specifically identify logs coming from BigQuery.

`LogType` is an enum that specifies the name and color of a log resource. It is used to understand what kind of log caused a change in the resource's state. In the UI, this is displayed as a colored label on the timeline (see `images/logtype.png` for an example). This is especially important when a resource's state can be determined by multiple different logs. For this BigQuery example, we only have the `jobCompletedEvent`, but the concept is critical for more complex scenarios.

**File:** `pkg/model/enum/log_type.go`

```go
// ...
const (
 // ...
 LogTypeControlPlaneComponent LogType = 12
 LogTypeSerialPort            LogType = 13
 LogTypeBigQueryResource      LogType = 14 // New
)

// ...

var LogTypes = map[LogType]LogTypeFrontendMetadata{
    // ...
 LogTypeSerialPort: {
  Label:                "serial_port",
  LabelBackgroundColor: "#333333",
 },
 // New
 LogTypeBigQueryResource: {
  EnumKeyName:          "LogTypeBigQueryResource",
  Label:                "bigquery_resource",
  LabelBackgroundColor: "#3fb549",
 },
}
```

### Step 3.2: Add new Revision States

Revision states represent the status of a resource at a point in time. For a BigQuery job, we need states for `Pending`, `Running`, `Success`, and `Failed`.

`RevisionState` is a critical enum that represents the "state" of a resource. It directly corresponds to the color of the bar in the timeline view (see `images/revision_state.png` for an example), making it essential for at-a-glance understanding of the resource's history.

**File:** `pkg/model/enum/revision_state.go`

```go
// ...
const (
 // ...
 RevisionStateProvisioning RevisionState = 29

 // BigQuery (New)
 RevisionStateBigQueryJobPending RevisionState = 30
 RevisionStateBigQueryJobRunning RevisionState = 31
 RevisionStateBigQueryJobSuccess RevisionState = 32
 RevisionStateBigQueryJobFailed  RevisionState = 33
)

// ...

var RevisionStates = map[RevisionState]RevisionStateFrontendMetadata{
    // ...
 // BigQuery (New)
 RevisionStateBigQueryJobPending: {
  EnumKeyName:     "RevisionStateBigQueryJobPending",
  BackgroundColor: "#997700",
  CSSSelector:     "job_pending",
  Label:           "Job is pending...",
 },
 RevisionStateBigQueryJobRunning: {
  EnumKeyName:     "RevisionStateBigQueryJobRunning",
  BackgroundColor: "#007700",
  CSSSelector:     "job_running",
  Label:           "Job is running...",
 },
 RevisionStateBigQueryJobSuccess: {
  EnumKeyName:     "RevisionStateBigQueryJobSuccess",
  BackgroundColor: "#113333",
  CSSSelector:     "job_success",
  Label:           "Job completed with success state",
 },
 RevisionStateBigQueryJobFailed: {
  EnumKeyName:     "RevisionStateBigQueryJobFailed",
  BackgroundColor: "#331111",
  CSSSelector:     "job_failed",
  Label:           "Job completed with errournous state",
 },
}
```

### Step 3.3: Add new Revision Verbs

Revision verbs describe the action that caused a state change. For our BigQuery job, the actions are `Create`, `Start`, and `Done`.

`RevisionVerb` is an enum that represents the trigger for a resource's state change. It provides context for *why* the state was altered.

**File:** `pkg/model/enum/verb.go`

```go
// ...
const (
 // ...
 RevisionVerbTerminating RevisionVerb = 31

 // BigQuery (New)
 RevisionVerbBigQuryJobCreate RevisionVerb = 32
 RevisionVerbBigQuryJobStart  RevisionVerb = 33
 RevisionVerbBigQuryJobDone   RevisionVerb = 34
)

// ...

var RevisionVerbs = map[RevisionVerb]RevisionVerbFrontendMetadata{
    // ...
 // New
 RevisionVerbBigQuryJobCreate: {
  EnumKeyName:          "RevisionVerbBigQuryJobCreate",
  Label:                "Create",
  LabelBackgroundColor: "#FDD835",
  CSSSelector:          "job-created",
 },
 RevisionVerbBigQuryJobStart: {
  EnumKeyName:          "RevisionVerbBigQuryJobStart",
  Label:                "Start",
  LabelBackgroundColor: "#22CC22",
  CSSSelector:          "job-started",
 },
 RevisionVerbBigQuryJobDone: {
  EnumKeyName:          "RevisionVerbBigQuryJobDone",
  Label:                "Done",
  LabelBackgroundColor: "#007700",
  CSSSelector:          "job-done",
 },
}
```

## Part 4: Implementing the BigQuery Tasks

This section covers the implementation of the tasks that power the BigQuery visualizer. KHI uses a Task System (see [`khi-task-system-concept.md`](../khi-task-system-concept.md) for details) to define a Directed Acyclic Graph (DAG) of operations. For this feature, the flow is:

1. **Query Task**: A task runs to generate a query that extracts the relevant BigQuery logs.
2. **Parser Task**: Another task takes the logs from the previous step and parses them.
3. **Event Registration**: The parser task transforms the log data into KHI's internal history model and registers the events.

We will implement this entire flow, starting with defining the data model.

With the enums in place, we can now implement the logic for fetching and parsing BigQuery logs. We'll create a new package for this: `pkg/task/googlecloudbigquery`.

> **Note on Package Structure**
> As recommended in the "Package structure for tasks" section of the [KHI Task system concept](../khi-task-system-concept.md) document, we are separating different concerns into sub-packages `googlecloudbigquery_contract` and `googlecloudbigquery_impl`.
> The contract package defines global constants like task IDs, inspection types and types used as the result of tasks. Any task implementation itself must not be defined in the contract folder.
> The impl package defines the actual implementations associated with the task IDs defined in contract. The impl must not be dependent from the other package.

### Step 4.1: Create Data Models

First, define the Go structs that will hold the data parsed from the logs. These structs should mirror the structure of the BigQuery job data in the audit logs.

**File:** `pkg/task/inspection/googlecloudbigquery/contract/model.go`

```go
package googlecloudbigquery_contract

import (
...
)

type BigQueryJob struct {
 Statistics BigQueryJobStatistics `yaml:"jobStatistics"`
 Name       BigqueryJobName       `yaml:"jobName"`
 Status     BigQueryJobStatus     `yaml:"jobStatus"`
}

func (b *BigQueryJob) ToResourcePath() resourcepath.ResourcePath {
 jobid := fmt.Sprintf("%s:%s.%s", b.Name.ProjectId, b.Name.Location, b.Name.JobId)
 return resourcepath.NameLayerGeneralItem("BigQuery", b.Statistics.Reservation, b.Name.ProjectId, jobid)
}

type BigQueryJobStatistics struct {
 CreateTime  string `yaml:"createTime"`
 StartTime   string `yaml:"startTime"`
 EndTime     string `yaml:"endTime"`
 Reservation string `yaml:"reservation"`
}

type BigqueryJobName struct {
 JobId     string `yaml:"jobId"`
 ProjectId string `yaml:"projectId"`
 Location  string `yaml:"location"`
}

type BigQueryJobStatus struct {
 State           string                `yaml:"state"`
 Error           BigQueryStatusError   `yaml:"error"`
 AdditionalError []BigQueryStatusError `yaml:"additionalErrors"`
}

type BigQueryStatusError struct {
 Code    int64  `yaml:"code"`
 Message string `yaml:"message"`
}
```

The `ToResourcePath` method is crucial. It creates a unique identifier for each BigQuery job, which KHI uses to group all related log entries.

### Step 4.2: Define Task IDs

Every task in KHI needs a unique ID. To keep the code organized and prevent import cycles, we define task IDs in a separate `taskid` package.

**File:** `pkg/task/inspection/googlecloudbigquery/contract/taskid.go`

```go
package googlecloudbigquery_contract

import (
 "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
 "github.com/GoogleCloudPlatform/khi/pkg/model/log"
 googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

var BigQueryTaskIDPrefix = googlecloudcommon_contract.GoogleCloudCommonTaskIDPrefix + "bigquery/"

var BigQueryCompletedEventQueryID = taskid.NewDefaultImplementationID[[]*log.Log](BigQueryTaskIDPrefix + "completedEvent")
var BigQueryJobParserTaskID = taskid.NewDefaultImplementationID[struct{}](BigQueryTaskIDPrefix + "jobs")

```

### Step 4.3: Implement the Log Query

This task generates the filter used to fetch the relevant audit logs from Google Cloud Logging. We'll place this in its own `query` package.

**File:** `pkg/task/inspection/googlecloudbigquery/impl/query_task.go`

```go
package googlecloudbigquery_impl

import (
 // Some imports...
)

func GenerateBigQueryJobCompletedFilter(projectId string) string {
 return fmt.Sprintf(`
resource.labels.project_id="%s"
resource.type="bigquery_resource"
protoPayload.serviceData.jobCompletedEvent.eventName="query_job_completed"
`, projectId)
}

var BigQueryJobCompletedTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudbigquery_contract.BigQueryCompletedEventQueryID, "BigQuery CompletedEvent logs", enum.LogTypeBigQueryResource, []taskid.UntypedTaskReference{
 googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
 projectId := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
 return []string{GenerateBigQueryJobCompletedFilter(projectId)}, nil
}, GenerateBigQueryJobCompletedFilter("google-cloud-project-id"))

```

### Step 4.4: Implement the Parser Logic

This is the core of our new feature. The parser takes a single log entry and creates multiple historical "revisions" from it. A BigQuery audit log for a completed job contains the create, start, and end times. Our parser creates a distinct event in the KHI timeline for each of these timestamps.

**File:** `pkg/task/inspection/googlecloudbigquery/impl/parser_task.go`

```go
package googlecloudbigquery_impl

import (
...
)

type bigqueryJobParser struct{}

// ... (other parser methods)

// LogTask implements parser.Parser.
func (b *bigqueryJobParser) LogTask() taskid.TaskReference[[]*log.Log] {
 return bqtaskid.BigQueryCompletedEventQueryID.Ref()
}

// Parse implements parser.Parser.
func (b *bigqueryJobParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
 job, err := NewBigQueryJobFromYamlStrings(l)
 if err != nil {
  return err
 }

 requester := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
 resourcePath := job.ToResourcePath()
 body, _ := l.Serialize("protoPayload.serviceData.jobCompletedEvent.job", &structured.YAMLNodeSerializer{})

 // Create a revision for the job's creation time
 cs.AddRevision(resourcePath, &history.StagingResourceRevision{
  Verb:       enum.RevisionVerbBigQuryJobCreate,
  State:      enum.RevisionStateBigQueryJobPending,
  Requestor:  requester,
  ChangeTime: parseTime(job.Statistics.CreateTime),
  Body:       string(body),
 })

 // Create a revision for the job's start time
 cs.AddRevision(resourcePath, &history.StagingResourceRevision{
  Verb:       enum.RevisionVerbBigQuryJobStart,
  State:      enum.RevisionStateBigQueryJobRunning,
  Requestor:  requester,
  ChangeTime: parseTime(job.Statistics.StartTime),
  Body:       string(body),
 })

 // Determine the final state (Success or Failed)
 isSuccess := job.Status.Error.Code == 0 && job.Status.Error.Message == ""
 var state enum.RevisionState
 if isSuccess {
  state = enum.RevisionStateBigQueryJobSuccess
 } else {
  state = enum.RevisionStateBigQueryJobFailed
 }

 // Create a revision for the job's end time
 cs.AddRevision(resourcePath, &history.StagingResourceRevision{
  Verb:       enum.RevisionVerbBigQuryJobDone,
  State:      state,
  Requestor:  requester,
  ChangeTime: parseTime(job.Statistics.EndTime),
  Body:       string(body),
 })
    
    // ...
 return nil
}

var BigQueryJobParserTask = parser.NewParserTaskFromParser(bqtaskid.BigQueryJobParserTaskID, &bigqueryJobParser{},1000, true, []string{bqinspectiontype.InspectionTypeId})
```

## Part 5: Testing the Feature

Testing is a critical part of the development process. KHI uses unit tests to ensure that each component works as expected.

### Step 5.1: Create Test Data

First, we need to create test data. This involves creating YAML files that represent the logs we want to test against. For our BigQuery parser, we need two files: one for a successful job and one for a failed job.

Create the directory `test/logs/bigquery/` and add the following files:

**File:** `test/logs/bigquery/single.yaml`

```yaml
# ... (content of the successful job log)
```

**File:** `test/logs/bigquery/internal_error.yaml`

```yaml
# ... (content of the failed job log)
```

*(You can get the full content of these files from the git history of the `bigquery` branch.)*

### Step 5.2: Write Unit Tests

Now, we can write the unit tests for our parser. The tests will read the YAML log files, run the parser, and then assert that the correct revisions were created.

**File:** `pkg/task/inspection/googlecloudbigquery/impl/parser_task_test.go`

```go
package googlecloudbigquery_impl

import (
 "testing"
 // ... imports
)

func TestParseSuccessBigQueryJob(t *testing.T) {
 cs, err := parser_test.ParseFromYamlLogFile(
  "test/logs/bigquery/single.yaml", // Test data for a successful job
  &bigqueryJobParser{},
  nil)
 if err != nil {
  t.Errorf("got error %v, want nil", err)
 }

 // ... assertions to verify the correct revisions are created
}

func TestParseFailedBVigQueryJob(t *testing.T) {
 cs, err := parser_test.ParseFromYamlLogFile(
  "test/logs/bigquery/internal_error.yaml", // Test data for a failed job
  &bigqueryJobParser{},
  nil)
 if err != nil {
  t.Errorf("got error %v, want nil", err)
 }

 // ... assertions to verify the final state is RevisionStateBigQueryJobFailed
}
```

### Step 5.3: Run the Tests

To run the tests, you can use the `make` command from the root of the project:

```bash
make test-go
```

This will run all the Go tests in the project, including the new ones you've just added.

## Part 6: Registering the New Feature

The final step is to register our new parser and inspection type with the KHI application so that it becomes available to users.

### Step 6.1: Define the Inspection Type

The inspection type tells KHI about your new feature, giving it a name, description, and icon to display in the UI. This should be in its own `inspectiontype` package.

**File:** `pkg/task/inspection/googlecloudbigquery/contract/inspection-type.go`

```go
package googlecloudbigquery_contract

import (
 "math"

 coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
)

var InspectionTypeId = "gcp-bigquery"

var BigQueryInspectionType = coreinspection.InspectionType{
 Id:          InspectionTypeId,
 Name:        "BigQuery",
 Description: `Visualize BigQuery Job. This inspection allows you to see all BigQuery jobs whitin the specified project.`,
 Icon:        "assets/icons/composer.webp",
 Priority:    math.MaxInt - 1001,
}
```

### Step 6.2: Register Tasks and Inspection Type

Finally, we register everything with the inspection server.
KHI build script automatically generates the code to call Register function defined in `registration.go` in the impl package.

**File:** `pkg/task/inspection/googlecloudbigquery/impl/registration.go`

```go
package googlecloudbigquery_impl

import (
...
)

// Register registers all googlecloudclustergdcbaremetal inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
 err := registry.AddInspectionType(googlecloudbigquery_contract.BigQueryInspectionType)
 if err != nil {
  return err
 }
 return coretask.RegisterTasks(registry,
  BigQueryJobCompletedTask,
  BigQueryJobParserTask,
 )
```

## Part 7: How the Frontend Uses This Data

While this feature is backend-only, it's helpful to understand how the frontend uses the data you've provided.

When the backend sends the history data to the frontend, it includes the `LogType`, `RevisionState`, and `RevisionVerb` for each event. The frontend uses this information to:

- **Display the correct colors and labels**: The `LogTypes`, `RevisionStates`, and `RevisionVerbs` maps in the `pkg/model/enum/` package are sent to the frontend, which uses them to render the correct colors and labels in the timeline.
- **Build the timeline**: The frontend iterates through the revisions for each resource and uses the `ChangeTime` to place them on the timeline. The `State` determines the color of the timeline bar at that point in time.

By adding the new enums, you've given the frontend all the information it needs to render the BigQuery job history correctly, without needing any frontend code changes.

## Conclusion

Congratulations! You have successfully added a new feature to KHI.

By following these steps, you have:

- Extended the core enums to support new concepts.
- Created a data model and a parser for a new log source.
- Written tests for your new feature.
- Registered the new tasks and inspection type, integrating your feature into the application.

This modular, task-based architecture is what makes KHI extensible. You can use this tutorial as a blueprint for adding your own custom log visualizers in the future.
