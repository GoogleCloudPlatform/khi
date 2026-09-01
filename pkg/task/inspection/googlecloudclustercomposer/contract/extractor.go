// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googlecloudclustercomposer_contract

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	pathTextPayload                 = structured.CompileFieldPath("textPayload")
	pathLabelsWorkerID              = structured.CompileFieldPath("labels.worker_id")
	pathLabelsSchedulerID           = structured.CompileFieldPath("labels.scheduler_id")
	pathLabelsDagProcessorManagerID = structured.CompileFieldPath("labels.dag_processor_manager_id")
	pathLabelsTriggererID           = structured.CompileFieldPath("labels.triggerer_id")
	pathLabelsWebserverID           = structured.CompileFieldPath("labels.webserver_id")
	pathLabelsSubservice            = structured.CompileFieldPath("labels.sub_service")
	pathLabelsPodID                 = structured.CompileFieldPath("labels.pod_id")
	pathLogName                     = structured.CompileFieldPath("logName")
	pathLabelsRunID                 = structured.CompileFieldPath("labels.run-id")
	pathLabelsWorkflow              = structured.CompileFieldPath("labels.workflow")
	pathLabelsTaskID                = structured.CompileFieldPath("labels.task-id")
	pathLabelsMapIndex              = structured.CompileFieldPath("labels.map-index")

	// \t<TaskInstance: $DAGID.$TASKID $RUNID map_index=$MAPINDEX [scheduled]>
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1179
	airflowTiTemplate = regexp.MustCompile(`\s<TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]>`)

	// Received executor event with state queued for task instance TaskInstanceKey(dag_id='khi_dag', task_id='add_one', run_id='scheduled__2023-11-30T05:00:00+00:00', try_number=1, map_index=0)
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L685
	airflowSchedulerReceivedEventTemplate = regexp.MustCompile(`Received executor event with state (?P<state>.+) for task instance TaskInstanceKey\(dag_id='(?P<dagid>.+)', task_id='(?P<taskid>.+)', run_id='(?P<runid>.+)',.*map_index=(?P<mapIndex>\d+)\)`)

	// TaskInstance Finished: dag_id=DAGID, task_id=TASKID, run_id=RUNID, map_index=MAPINDEX, ..., state=STATE ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L715
	airflowSchedulerTaskFinishedTemplate = regexp.MustCompile(`TaskInstance Finished:\s+dag_id=(?P<dagid>\S+),\s+task_id=(?P<taskid>\S+),\s+run_id=(?P<runid>\S+),\s+map_index=(?P<mapIndex>\S+),\s+.*?state=(?P<state>\S+)(?:,\s+executor=.+?)?,\s+executor_state.+`)

	// Detected zombie job: {'full_filepath': '...', 'processor_subdir': '...', 'msg': "{'DAG Id': 'DAG_ID', 'Task Id': 'TASK_ID', 'Run Id': 'RUN_ID', 'Hostname': 'WORKER', ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L1746C55-L1746C62
	airflowSchedulerZombieDetectedTemplate = regexp.MustCompile(`'DAG Id':\s*'(?P<dagid>[^']+)',\s*'Task Id':\s*'(?P<taskid>[^']+)',\s*'Run Id':\s*'(?P<runid>[^']+)',\s*('Map Index':\s*'(?P<mapIndex>[^']+)',\s*)?'Hostname':\s*'(?P<host>[^']+)'`)

	// Running <TaskInstance: DAG_ID.TASK_ID RUN_ID [STATE]> on host WORKER
	airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)

	// Marking task as STATE. dag_id=DAG_ID, task_id=TASK_ID, run_id=RUN_ID, map_index=MAP_INDEX, execution_date=..., start_date=..., end_date=...
	airflowWorkerMarkingStatusTemplate = regexp.MustCompile(`.*Marking task as\s(?P<state>\S+).\sdag_id=(?P<dagid>\S+),\stask_id=(?P<taskid>\S+),\srun_id=(?P<runid>\S+),\s(map_index=(?P<mapIndex>\d+),\s)?.+`)

	// Task finished [task_instance_id=019d10e4-71f5-7016-b412-aa1fbcfd16fc] [exit_code=0] [duration=0.41656468300061533] [final_state=skipped]
	airflowWorkerFinalStateExtractTemplate = regexp.MustCompile(`\[final_state=(?P<state>[a-z_]*)\]`)
)

// ComposerTaskInstanceFieldSet contains parsed Airflow task instance info.
type ComposerTaskInstanceFieldSet struct {
	TaskInstance *AirflowTaskInstance
}

// ComposerFieldSet contains component identity info for Composer logs.
type ComposerFieldSet struct {
	Component             string // e.g. "worker", "scheduler", "dag-processor-manager"
	WorkerID              string
	SchedulerID           string
	DagProcessorManagerID string
	TriggererID           string
	WebserverID           string
	Subservice            string
}

// ComposerWorkerTaskInstanceFieldSet contains worker-specific task instance info.
type ComposerWorkerTaskInstanceFieldSet struct {
	TaskInstance *AirflowTaskInstance
}

func stringToTiState(stateStr string) (Tistate, error) {
	switch strings.ToLower(stateStr) {
	case "scheduled":
		return TASKINSTANCE_SCHEDULED, nil
	case "queued":
		return TASKINSTANCE_QUEUED, nil
	case "running":
		return TASKINSTANCE_RUNNING, nil
	case "success":
		return TASKINSTANCE_SUCCESS, nil
	case "failed":
		return TASKINSTANCE_FAILED, nil
	case "deferred":
		return TASKINSTANCE_DEFERRED, nil
	case "up_for_retry":
		return TASKINSTANCE_UP_FOR_RETRY, nil
	case "up_for_reschedule":
		return TASKINSTANCE_UP_FOR_RESCHEDULE, nil
	case "removed":
		return TASKINSTANCE_REMOVED, nil
	case "upstream_failed":
		return TASKINSTANCE_UPSTREAM_FAILED, nil
	case "zombie":
		return TASKINSTANCE_ZOMBIE, nil
	case "skipped":
		return TASKINSTANCE_SKIPPED, nil
	default:
		return "", fmt.Errorf("unknown Airflow task state: %s", stateStr)
	}
}

// ExtractComposer extracts ComposerFieldSet from a NodeReader.
func ExtractComposer(reader *structured.NodeReader) (ComposerFieldSet, error) {
	if mock, ok := structured.GetMock[ComposerFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil {
		return ComposerFieldSet{}, nil
	}

	workerID := reader.ReadStringOrDefault(pathLabelsWorkerID, "")
	schedulerID := reader.ReadStringOrDefault(pathLabelsSchedulerID, "")
	dagProcessorManagerID := reader.ReadStringOrDefault(pathLabelsDagProcessorManagerID, "")
	triggererID := reader.ReadStringOrDefault(pathLabelsTriggererID, "")
	webserverID := reader.ReadStringOrDefault(pathLabelsWebserverID, "")
	subservice := reader.ReadStringOrDefault(pathLabelsSubservice, "")
	podID := reader.ReadStringOrDefault(pathLabelsPodID, "")
	logName := reader.ReadStringOrDefault(pathLogName, "")

	componentNameIndex := strings.LastIndex(logName, "/")
	if componentNameIndex == -1 {
		return ComposerFieldSet{}, fmt.Errorf("not a recognized composer component log")
	}
	component := logName[componentNameIndex+1:]

	if component == "" {
		return ComposerFieldSet{}, fmt.Errorf("not a recognized composer component log")
	}

	if podID != "" {
		switch {
		case strings.HasPrefix(podID, "airflow-worker-"):
			workerID = podID
		case strings.HasPrefix(podID, "airflow-scheduler-"):
			schedulerID = podID
		case strings.HasPrefix(podID, "airflow-dag-processor-manager-"):
			dagProcessorManagerID = podID
		case strings.HasPrefix(podID, "airflow-triggerer-"):
			triggererID = podID
		case strings.HasPrefix(podID, "airflow-webserver-"):
			webserverID = podID
		}
	}

	if subservice == "" {
		subservice = component
	}

	return ComposerFieldSet{
		Component:             component,
		WorkerID:              workerID,
		SchedulerID:           schedulerID,
		DagProcessorManagerID: dagProcessorManagerID,
		TriggererID:           triggererID,
		WebserverID:           webserverID,
		Subservice:            subservice,
	}, nil
}

// ExtractComposerTaskInstance extracts ComposerTaskInstanceFieldSet from a NodeReader.
func ExtractComposerTaskInstance(reader *structured.NodeReader) (ComposerTaskInstanceFieldSet, error) {
	if mock, ok := structured.GetMock[ComposerTaskInstanceFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil {
		return ComposerTaskInstanceFieldSet{}, fmt.Errorf("nil reader")
	}

	textPayload, err := reader.ReadString(pathTextPayload)
	if err != nil {
		return ComposerTaskInstanceFieldSet{}, fmt.Errorf("textPayload not found: %w", err)
	}

	template := []*regexp.Regexp{
		airflowTiTemplate,
		airflowSchedulerReceivedEventTemplate,
		airflowSchedulerTaskFinishedTemplate,
	}

	for _, re := range template {
		matches := re.FindStringSubmatch(textPayload)
		if matches == nil {
			continue
		}
		dagid := matches[re.SubexpIndex("dagid")]
		taskid := matches[re.SubexpIndex("taskid")]
		runid := matches[re.SubexpIndex("runid")]
		stateStr := matches[re.SubexpIndex("state")]
		mapIndex := "-1"
		if i := re.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
			mapIndex = matches[i]
		}
		state, err := stringToTiState(stateStr)
		if err != nil {
			continue
		}
		return ComposerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, "", state),
		}, nil
	}

	matches := airflowSchedulerZombieDetectedTemplate.FindStringSubmatch(textPayload)
	if matches != nil {
		dagid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("dagid")]
		taskid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("taskid")]
		runid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("runid")]
		state := TASKINSTANCE_ZOMBIE
		host := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("host")]
		mapIndex := "-1"
		if i := airflowSchedulerZombieDetectedTemplate.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
			mapIndex = matches[i]
		}
		return ComposerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state),
		}, nil
	}

	return ComposerTaskInstanceFieldSet{}, fmt.Errorf("not an Airflow TaskInstance log")
}

type workerTaskInstanceInfo struct {
	dagId    string
	taskId   string
	runId    string
	mapIndex string
	workerId string
	state    Tistate
}

func (i workerTaskInstanceInfo) merge(other workerTaskInstanceInfo) workerTaskInstanceInfo {
	res := i
	if res.dagId == "" {
		res.dagId = other.dagId
	}
	if res.taskId == "" {
		res.taskId = other.taskId
	}
	if res.runId == "" {
		res.runId = other.runId
	}
	if (res.mapIndex == "" || res.mapIndex == "-1") && other.mapIndex != "" {
		res.mapIndex = other.mapIndex
	}
	if other.workerId != "" {
		res.workerId = other.workerId
	}
	if other.state != TASKINSTANCE_NONE && other.state != "" {
		res.state = other.state
	}
	return res
}

func readWorkerLabels(reader *structured.NodeReader) workerTaskInstanceInfo {
	workerID := reader.ReadStringOrDefault(pathLabelsWorkerID, "")
	runID := reader.ReadStringOrDefault(pathLabelsRunID, "")
	workflow := reader.ReadStringOrDefault(pathLabelsWorkflow, "")
	taskID := reader.ReadStringOrDefault(pathLabelsTaskID, "")
	mapIndex := reader.ReadStringOrDefault(pathLabelsMapIndex, "")

	return workerTaskInstanceInfo{
		dagId:    workflow,
		taskId:   taskID,
		runId:    runID,
		mapIndex: mapIndex,
		workerId: workerID,
		state:    TASKINSTANCE_NONE,
	}
}

func parseWorkerPayload(textPayload string) workerTaskInstanceInfo {
	if strings.HasPrefix(textPayload, "Running ") {
		if matches := airflowWorkerRunningHostTemplate.FindStringSubmatch(textPayload); matches != nil {
			mapIndex := ""
			if i := airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
				mapIndex = matches[i]
			}
			stateStr := matches[airflowWorkerRunningHostTemplate.SubexpIndex("state")]
			state, err := stringToTiState(stateStr)
			if err != nil {
				slog.Warn(fmt.Sprintf("failed to parse task instance state %q: %v", stateStr, err))
			}
			return workerTaskInstanceInfo{
				dagId:    matches[airflowWorkerRunningHostTemplate.SubexpIndex("dagid")],
				taskId:   matches[airflowWorkerRunningHostTemplate.SubexpIndex("taskid")],
				runId:    matches[airflowWorkerRunningHostTemplate.SubexpIndex("runid")],
				workerId: matches[airflowWorkerRunningHostTemplate.SubexpIndex("host")],
				mapIndex: mapIndex,
				state:    state,
			}
		}
	}
	if matches := airflowWorkerMarkingStatusTemplate.FindStringSubmatch(textPayload); matches != nil {
		mapIndex := ""
		if i := airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
			mapIndex = matches[i]
		}
		stateStr := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("state")]
		state, err := stringToTiState(stateStr)
		if err != nil {
			slog.Warn(fmt.Sprintf("failed to parse task instance state %q: %v", stateStr, err))
		}
		return workerTaskInstanceInfo{
			dagId:    matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("dagid")],
			taskId:   matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("taskid")],
			runId:    matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("runid")],
			mapIndex: mapIndex,
			state:    state,
		}
	}
	if matches := airflowWorkerFinalStateExtractTemplate.FindStringSubmatch(textPayload); matches != nil {
		stateStr := matches[airflowWorkerFinalStateExtractTemplate.SubexpIndex("state")]
		state, err := stringToTiState(stateStr)
		if err != nil {
			slog.Warn(fmt.Sprintf("failed to parse task instance state %q: %v", stateStr, err))
		}
		return workerTaskInstanceInfo{
			state: state,
		}
	}
	return workerTaskInstanceInfo{}
}

// ExtractComposerWorkerTaskInstance extracts ComposerWorkerTaskInstanceFieldSet from a NodeReader.
func ExtractComposerWorkerTaskInstance(reader *structured.NodeReader) (ComposerWorkerTaskInstanceFieldSet, error) {
	if mock, ok := structured.GetMock[ComposerWorkerTaskInstanceFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil {
		return ComposerWorkerTaskInstanceFieldSet{}, fmt.Errorf("nil reader")
	}

	labelInfo := readWorkerLabels(reader)

	textPayload, err := reader.ReadString(pathTextPayload)
	if err != nil {
		return ComposerWorkerTaskInstanceFieldSet{}, fmt.Errorf("textPayload not found: %w", err)
	}

	payloadInfo := parseWorkerPayload(textPayload)
	info := labelInfo.merge(payloadInfo)

	if info.mapIndex == "" {
		info.mapIndex = "-1"
	}

	if info.dagId == "" || info.taskId == "" || info.runId == "" || info.workerId == "" {
		return ComposerWorkerTaskInstanceFieldSet{}, fmt.Errorf("not an Airflow Worker TaskInstance log")
	}

	return ComposerWorkerTaskInstanceFieldSet{
		TaskInstance: NewAirflowTaskInstance(info.dagId, info.taskId, info.runId, info.mapIndex, info.workerId, info.state),
	}, nil
}
