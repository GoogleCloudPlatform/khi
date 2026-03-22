// Copyright 2025 Google LLC
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
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type ComposerTaskInstanceFieldSet struct {
	TaskInstance *AirflowTaskInstance
}

func (c *ComposerTaskInstanceFieldSet) Kind() string {
	return "ComposerTaskInstance"
}

var _ log.FieldSet = &ComposerTaskInstanceFieldSet{}

type ComposerTaskInstanceFieldSetReader struct{}

func (c *ComposerTaskInstanceFieldSetReader) FieldSetKind() string {
	return (&ComposerTaskInstanceFieldSet{}).Kind()
}

var (
	// \t<TaskInstance: $DAGID.$TASKID $RUNID map_index=$MAPINDEX [scheduled]>
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1179
	airflowTiTemplate = regexp.MustCompile(`\s<TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]>`)

	// TODO Add log types
	// * Trying to enqueue tasks: [<TaskInstance: airflow_monitoring.echo scheduled__2025-04-10T04:00:00+00:00 [scheduled]>] for executor: CeleryExecutor(parallelism=0) (ONLY appliucable from 2.10.x)
	// * Sending TaskInstanceKey(dag_id='airflow_monitoring', task_id='echo', run_id='scheduled__2025-04-10T04:00:00+00:00', try_number=1, map_index=-1) to CeleryExecutor with priority 2147483647 and queue default
	// * Adding to queue: ['airflow', 'tasks', 'run', 'airflow_monitoring', 'echo', 'scheduled__2025-04-10T04:00:00+00:00', '--local', '--subdir', 'DAGS_FOLDER/airflow_monitoring.py']

	// Received executor event with state queued for task instance TaskInstanceKey(dag_id='khi_dag', task_id='add_one', run_id='scheduled__2023-11-30T05:00:00+00:00', try_number=1, map_index=0)
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L685
	airflowSchedulerReceivedEventTemplate = regexp.MustCompile(`Received executor event with state (?P<state>.+) for task instance TaskInstanceKey\(dag_id='(?P<dagid>.+)', task_id='(?P<taskid>.+)', run_id='(?P<runid>.+)',.*map_index=(?P<mapIndex>\d+)\)`)

	// TODO Add other log types
	// * Setting external_id for <TaskInstance: airflow_monitoring.echo scheduled__2025-04-10T04:00:00+00:00 [queued]> to cf33ab13-b638-4abb-8484-9faf4cc19345
	// * Marking run <DagRun airflow_monitoring @ 2025-04-10 04:00:00+00:00: scheduled__2025-04-10T04:00:00+00:00, state:running, queued_at: 2025-04-10 04:10:00.679237+00:00. externally triggered: False> successful

	// TaskInstance Finished: dag_id=DAGID, task_id=TASKID, run_id=RUNID, map_index=MAPINDEX, ..., state=STATE ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L715
	airflowSchedulerTaskFinishedTemplate = regexp.MustCompile(`TaskInstance Finished:\s+dag_id=(?P<dagid>\S+),\s+task_id=(?P<taskid>\S+),\s+run_id=(?P<runid>\S+),\s+map_index=(?P<mapIndex>\S+),\s+.*?state=(?P<state>\S+)(?:,\s+executor=.+?)?,\s+executor_state.+`)

	// TODO Add other log types
	// * Received executor event with state success for task instance TaskInstanceKey(dag_id='airflow_monitoring', task_id='echo', run_id='scheduled__2025-04-10T04:00:00+00:00', try_number=1, map_index=-1)

	// Detected zombie job: {'full_filepath': '...', 'processor_subdir': '...', 'msg': "{'DAG Id': 'DAG_ID', 'Task Id': 'TASK_ID', 'Run Id': 'RUN_ID', 'Hostname': 'WORKER', ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L1746C55-L1746C62
	airflowSchedulerZombieDetectedTemplate = regexp.MustCompile(`'DAG Id':\s*'(?P<dagid>[^']+)',\s*'Task Id':\s*'(?P<taskid>[^']+)',\s*'Run Id':\s*'(?P<runid>[^']+)',\s*('Map Index':\s*'(?P<mapIndex>[^']+)',\s*)?'Hostname':\s*'(?P<host>[^']+)'`)
)

func stringToTiState(stateStr string) (Tistate, error) {
	switch stateStr {
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

func (c *ComposerTaskInstanceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	textPayload, err := reader.ReadString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found")
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
		return &ComposerTaskInstanceFieldSet{
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
		return &ComposerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state),
		}, nil
	}

	return nil, fmt.Errorf("not an Airflow TaskInstance log")
}

var _ log.FieldSetReader = &ComposerTaskInstanceFieldSetReader{}

type ComposerFieldSet struct {
	Component             string // e.g. "worker", "scheduler", "dag-processor-manager"
	WorkerID              string
	SchedulerID           string
	DagProcessorManagerID string
	TriggererID           string
	WebserverID           string
	Subservice            string
}

func (c *ComposerFieldSet) Kind() string {
	return "Composer"
}

var _ log.FieldSet = &ComposerFieldSet{}

type ComposerFieldSetReader struct{}

func (c *ComposerFieldSetReader) FieldSetKind() string {
	return (&ComposerFieldSet{}).Kind()
}

func (c *ComposerFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	workerId, _ := reader.ReadString("labels.worker_id")
	schedulerId, _ := reader.ReadString("labels.scheduler_id")
	dagProcessorManagerId, _ := reader.ReadString("labels.dag_processor_manager_id")
	triggererId, _ := reader.ReadString("labels.triggerer_id")
	webserverId, _ := reader.ReadString("labels.webserver_id")
	subservice, _ := reader.ReadString("labels.sub_service")
	podId, _ := reader.ReadString("labels.pod_id")
	logName, _ := reader.ReadString("logName")

	componentNameIndex := strings.LastIndex(logName, "/")
	if componentNameIndex == -1 {
		return nil, fmt.Errorf("not a recognized composer component log")
	}
	component := logName[componentNameIndex+1:]

	if component == "" {
		return nil, fmt.Errorf("not a recognized composer component log")
	}

	if podId != "" {
		switch {
		case strings.HasPrefix(podId, "airflow-worker-"):
			workerId = podId
		case strings.HasPrefix(podId, "airflow-scheduler-"):
			schedulerId = podId
		case strings.HasPrefix(podId, "airflow-dag-processor-manager-"):
			dagProcessorManagerId = podId
		case strings.HasPrefix(podId, "airflow-triggerer-"):
			triggererId = podId
		case strings.HasPrefix(podId, "airflow-webserver-"):
			webserverId = podId
		}
	}

	if subservice == "" {
		subservice = component
	}

	return &ComposerFieldSet{
		Component:             component,
		WorkerID:              workerId,
		SchedulerID:           schedulerId,
		DagProcessorManagerID: dagProcessorManagerId,
		TriggererID:           triggererId,
		WebserverID:           webserverId,
		Subservice:            subservice,
	}, nil
}

var _ log.FieldSetReader = &ComposerFieldSetReader{}

type ComposerWorkerTaskInstanceFieldSet struct {
	TaskInstance *AirflowTaskInstance
}

func (c *ComposerWorkerTaskInstanceFieldSet) Kind() string {
	return "ComposerWorkerTaskInstance"
}

var _ log.FieldSet = &ComposerWorkerTaskInstanceFieldSet{}

type ComposerWorkerTaskInstanceFieldSetReader struct{}

func (c *ComposerWorkerTaskInstanceFieldSetReader) FieldSetKind() string {
	return (&ComposerWorkerTaskInstanceFieldSet{}).Kind()
}

var (
	// Running <TaskInstance: DAG_ID.TASK_ID RUN_ID [STATE]> on host WORKER
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/cli/commands/task_command.py#L416
	// airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\S+)\.(?P<taskid>\S+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)
	airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)

	// Marking task as STATE. dag_id=DAG_ID, task_id=TASK_ID, run_id=RUN_ID, map_index=MAP_INDEX, execution_date=..., start_date=..., end_date=...
	// ref: https://github.com/apache/airflow/blob/2.9.3/airflow/models/taskinstance.py#L1201
	airflowWorkerMarkingStatusTemplate = regexp.MustCompile(`.*Marking task as\s(?P<state>\S+).\sdag_id=(?P<dagid>\S+),\stask_id=(?P<taskid>\S+),\srun_id=(?P<runid>\S+),\s(map_index=(?P<mapIndex>\d+),\s)?.+`)

	// Task finished [task_instance_id=019d10e4-71f5-7016-b412-aa1fbcfd16fc] [exit_code=0] [duration=0.41656468300061533] [final_state=skipped]
	// Airflow worker logs at finishing a task execution.
	// TODO: extract structured message field for airflow with more robust method.
	airflowWorkerFinalStateExtractTemplate = regexp.MustCompile(`\[final_state=(?P<state>[a-z_]*)\]`)
)

func (c *ComposerWorkerTaskInstanceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	if ti, err := c.tryReadFromLabels(reader); err == nil {
		return &ComposerWorkerTaskInstanceFieldSet{TaskInstance: ti}, nil
	}
	textPayload, err := reader.ReadString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found")
	}

	workerId, _ := reader.ReadString("labels.worker_id")

	if strings.HasPrefix(textPayload, "Running ") {
		matches := airflowWorkerRunningHostTemplate.FindStringSubmatch(textPayload)
		if matches != nil {
			dagid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("dagid")]
			taskid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("taskid")]
			runid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("runid")]
			host := matches[airflowWorkerRunningHostTemplate.SubexpIndex("host")]
			stateStr := matches[airflowWorkerRunningHostTemplate.SubexpIndex("state")]
			state, err := stringToTiState(stateStr)
			if err != nil {
				return nil, err
			}
			mapIndex := "-1"
			if i := airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
				mapIndex = matches[i]
			}
			return &ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state),
			}, nil
		}
	}

	matches := airflowWorkerMarkingStatusTemplate.FindStringSubmatch(textPayload)
	if matches != nil {
		if workerId == "" {
			return nil, fmt.Errorf("worker_id not found")
		}

		dagid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("dagid")]
		taskid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("taskid")]
		runid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("runid")]
		// Need strings.ToLower because it might be capitalized depending on the version
		stateStr := strings.ToLower(matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("state")])
		state, err := stringToTiState(stateStr)
		if err != nil {
			return nil, err
		}
		mapIndex := "-1"
		if i := airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex"); i >= 0 && matches[i] != "" {
			mapIndex = matches[i]
		}
		return &ComposerWorkerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, workerId, state),
		}, nil
	}

	return nil, fmt.Errorf("not an Airflow Worker TaskInstance log")
}

// tryReadFromLabels reads task instance info from labels if available.
// This is effective only when airflow 3.x is used.
func (c *ComposerWorkerTaskInstanceFieldSetReader) tryReadFromLabels(reader *structured.NodeReader) (*AirflowTaskInstance, error) {
	workerId, err := reader.ReadString("labels.worker_id")
	if err != nil {
		return nil, fmt.Errorf("worker_id not found")
	}
	runid, err := reader.ReadString("labels.run-id")
	if err != nil {
		return nil, fmt.Errorf("run-id not found")
	}

	workflow, err := reader.ReadString("labels.workflow")
	if err != nil {
		return nil, fmt.Errorf("workflow not found")
	}

	taskid, err := reader.ReadString("labels.task-id")
	if err != nil {
		return nil, fmt.Errorf("task-id not found")
	}

	mapIndex, err := reader.ReadString("labels.map-index")
	if err != nil {
		return nil, fmt.Errorf("map-index not found")
	}

	textPayload, err := reader.ReadString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found")
	}

	matches := airflowWorkerFinalStateExtractTemplate.FindStringSubmatch(textPayload)
	state := TASKINSTANCE_NONE
	if matches != nil {
		stateStr := strings.ToLower(matches[airflowWorkerFinalStateExtractTemplate.SubexpIndex("state")])
		if finalState, err := stringToTiState(stateStr); err == nil {
			state = finalState
		}
	}

	return NewAirflowTaskInstance(workflow, taskid, runid, mapIndex, workerId, state), nil
}

var _ log.FieldSetReader = &ComposerWorkerTaskInstanceFieldSetReader{}
