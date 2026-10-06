# Cloud Composer Airflow Inspection Tasks

This package (`googlecloud/composerairflow`) and its cluster definition counterpart (`googlecloud/cluster/composer`) contain the tasks for inspecting Google Cloud Composer environments. It performs environment discovery, log fetching, filtering, parsing, and mapping into Kubernetes History Inspector (KHI) timeline events.

## Task Overview

The Composer inspection pipeline can be divided into four main phases:

1. **Discovery & Inputs**: Resolving the target Composer environment and the components the user wants to inspect.
2. **Log Fetching**: Querying Cloud Logging for the target log entries.
3. **Parsing & Mapping Pipelines**: Reading log fields and mapping them to KHI timeline events. The pipeline splits into parallel streams depending on the Airflow component (Scheduler, Worker, DAG Processor Manager, or Other fallback).
4. **Aggregation**: Unifying the pipelines into a final feature task sequence.

### 1. Discovery & Inputs

- **`autocompleteComposerEnvironmentIdentityTask`**: Suggests available Composer environments.
- **`autocompleteLocationForComposerEnvironmentTask`**: Suggests the location of the selected environment.
- **`inputComposerEnvironmentNameTask`**: Captures the user-selected environment name.
- **`autocompleteComposerComponentsTask`**: Queries Cloud Monitoring (`logging.googleapis.com/log_entry_count`) to dynamically suggest available Airflow components (e.g. `scheduler`, `worker`, `dag-processor-manager`, `webserver`, etc.).
- **`inputComposerComponentsTask`**: Captures the user-selected components to inspect.

### 2. Log Fetching

- **`composerLogsQueryTask`**: Generates the Cloud Logging query based on input properties and fetches the raw logs.

### 3. Parsing & Mapping Pipelines

Logs are filtered into specific component streams using extractors. Each stream typically follows the pattern of:
`Filter` -> `Grouper` -> `Ingester` -> `Mapper`

- **Scheduler Pipeline**: Handles `airflow-scheduler` component logs.
  - Tasks: `airflowSchedulerLogFilterTask`, `airflowSchedulerLogGrouperTask`, `airflowSchedulerLogIngesterTask`, `airflowSchedulerLogToTimelineMapperTask`.
- **Worker Pipeline**: Handles `airflow-worker` component logs.
  - Tasks: `airflowWorkerLogFilterTask`, `airflowWorkerLogGrouperTask`, `airflowWorkerLogIngesterTask`, `airflowWorkerLogToTimelineMapperTask`.
- **Dag Processor Manager Pipeline**: Handles `airflow-dag-processor-manager` logs.
  - Tasks: `airflowDagProcessorManagerLogFilterTask`, `airflowDagProcessorManagerLogGrouperTask`, `airflowDagProcessorManagerLogIngesterTask`, `airflowDagProcessorManagerLogToTimelineMapperTask`.
- **Other Pipeline (Fallback)**: Catches any component logs that do not match the above three (e.g., `webserver`, `triggerer`).
  - Tasks: `airflowOtherLogFilterTask`, `airflowOtherLogGrouperTask`, `airflowOtherLogIngesterTask`, `airflowOtherLogToTimelineMapperTask`.

### 4. Aggregation

- **`composerLogsTailTask`**: Waits for all `...LogToTimelineMapperTask` tasks to complete and exposes the Composer logs feature toggle.

## Task Relationship Diagram

The following Mermaid diagram illustrates the dependencies and data flow through the Composer inspection tasks.

```mermaid
graph TD
    classDef input fill:#e0f7fa,stroke:#006064,stroke-width:2px;
    classDef query fill:#fff3e0,stroke:#e65100,stroke-width:2px;
    classDef pipeline fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px;
    classDef tail fill:#ede7f6,stroke:#4527a0,stroke-width:2px;

    classDef external fill:#e0f7fa,stroke:#33691e,stroke-width:2px,stroke-dasharray: 5 5;

    %% External Tasks
    ProjectIDInput[inputProjectIdTask]:::external
    LocationInput[inputLocationsTask]:::external
    StartTime[inputStartTimeTask]:::external
    EndTime[inputEndTimeTask]:::external
    ClusterIdentity[clusterIdentityTask]:::pipeline

    %% Composer Discovery & Input
    EnvIdentityAuto[autocompleteComposerEnvironmentIdentityTask]:::pipeline
    LocationAuto[autocompleteLocationForComposerEnvironmentTask]:::pipeline
    EnvInput[inputComposerEnvironmentNameTask]:::input
    CompAuto[autocompleteComposerComponentsTask]:::pipeline
    CompInput[inputComposerComponentsTask]:::input

    %% Dependencies for AutocompleteComposerEnvironmentIdentity
    ProjectIDInput --> EnvIdentityAuto
    StartTime --> EnvIdentityAuto
    EndTime --> EnvIdentityAuto

    %% Dependencies for InputComposerEnvironmentName
    EnvIdentityAuto --> EnvInput

    %% Dependencies for AutocompleteLocation
    ProjectIDInput --> LocationAuto
    EnvInput --> LocationAuto
    StartTime --> LocationAuto
    EndTime --> LocationAuto
    LocationAuto --> LocationInput


    %% ClusterIdentity depends on project and location
    ProjectIDInput --> ClusterIdentity
    LocationInput --> ClusterIdentity

    %% Dependencies for AutocompleteComposerComponents
    ClusterIdentity --> CompAuto
    StartTime --> CompAuto
    EndTime --> CompAuto
    EnvInput --> CompAuto

    %% Dependencies for InputComposerComponents
    CompAuto --> CompInput

    %% Log Fetching
    ClusterIdentity --> LogQuery[composerLogsQueryTask]:::query
    EnvInput --> LogQuery
    CompInput --> LogQuery

    %% Pipelines
    LogQuery --> SchedFilter[airflowSchedulerLogFilterTask]:::pipeline
    SchedFilter --> SchedGrouper[airflowSchedulerLogGrouperTask]:::pipeline
    SchedGrouper --> SchedIngester[airflowSchedulerLogIngesterTask]:::pipeline
    SchedIngester --> SchedMapper[airflowSchedulerLogToTimelineMapperTask]:::pipeline

    LogQuery --> WorkFilter[airflowWorkerLogFilterTask]:::pipeline
    WorkFilter --> WorkGrouper[airflowWorkerLogGrouperTask]:::pipeline
    WorkGrouper --> WorkIngester[airflowWorkerLogIngesterTask]:::pipeline
    WorkIngester --> WorkMapper[airflowWorkerLogToTimelineMapperTask]:::pipeline

    LogQuery --> DpmFilter[airflowDagProcessorManagerLogFilterTask]:::pipeline
    DpmFilter --> DpmGrouper[airflowDagProcessorManagerLogGrouperTask]:::pipeline
    DpmGrouper --> DpmIngester[airflowDagProcessorManagerLogIngesterTask]:::pipeline
    DpmIngester --> DpmMapper[airflowDagProcessorManagerLogToTimelineMapperTask]:::pipeline

    LogQuery --> OtherFilter[airflowOtherLogFilterTask]:::pipeline
    OtherFilter --> OtherGrouper[airflowOtherLogGrouperTask]:::pipeline
    OtherGrouper --> OtherIngester[airflowOtherLogIngesterTask]:::pipeline
    OtherIngester --> OtherMapper[airflowOtherLogToTimelineMapperTask]:::pipeline

    %% Aggregation
    SchedMapper --> TailTask[composerLogsTailTask]:::tail
    WorkMapper --> TailTask
    DpmMapper --> TailTask
    OtherMapper --> TailTask
```
