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
	airflowTiTemplate                      = regexp.MustCompile(`TaskInstance:\s+\[<TaskInstance:\s+(?P<dagid>\S+)\.(?P<taskid>\S+)\s+(?P<runid>\S+)(\s+map_index=(?P<mapIndex>\d+))?\s+\[(?P<state>\S+)\]>\]`)
	airflowSchedulerReceivedEventTemplate  = regexp.MustCompile(`Received executor event with state (?P<state>\S+) for task instance TaskInstanceKey\(dag_id='(?P<dagid>[^']+)', task_id='(?P<taskid>[^']+)', run_id='(?P<runid>[^']+)', try_number=\d+, map_index=(?P<mapIndex>-?\d+)\)`)
	airflowSchedulerTaskFinishedTemplate   = regexp.MustCompile(`TaskInstance Finished:\s+dag_id=(?P<dagid>[^,]+),\s+task_id=(?P<taskid>[^,]+),\s+run_id=(?P<runid>[^,]+),\s+map_index=(?P<mapIndex>-?\d+),.*state=(?P<state>[^,]+),`)
	airflowSchedulerZombieDetectedTemplate = regexp.MustCompile(`Detected zombie job:.*'DAG Id':\s+'(?P<dagid>[^']+)'.*'Task Id':\s+'(?P<taskid>[^']+)'.*'Run Id':\s+'(?P<runid>[^']+)'.*'Hostname':\s+'(?P<host>[^']+)'`)
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

type ComposerSchedulerFieldSet struct {
	SchedulerID string
}

func (c *ComposerSchedulerFieldSet) Kind() string {
	return "ComposerScheduler"
}

var _ log.FieldSet = &ComposerSchedulerFieldSet{}

type ComposerSchedulerFieldSetReader struct{}

func (c *ComposerSchedulerFieldSetReader) FieldSetKind() string {
	return (&ComposerSchedulerFieldSet{}).Kind()
}

func (c *ComposerSchedulerFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	schedulerId, err := reader.ReadString("labels.scheduler_id")
	if err != nil || schedulerId == "" {
		return nil, fmt.Errorf("scheduler_id not found")
	}

	return &ComposerSchedulerFieldSet{
		SchedulerID: schedulerId,
	}, nil
}

var _ log.FieldSetReader = &ComposerSchedulerFieldSetReader{}

type ComposerWorkerFieldSet struct {
	WorkerID string
}

func (c *ComposerWorkerFieldSet) Kind() string {
	return "ComposerWorker"
}

var _ log.FieldSet = &ComposerWorkerFieldSet{}

type ComposerWorkerFieldSetReader struct{}

func (c *ComposerWorkerFieldSetReader) FieldSetKind() string {
	return (&ComposerWorkerFieldSet{}).Kind()
}

func (c *ComposerWorkerFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	workerId, err := reader.ReadString("labels.worker_id")
	if err != nil || workerId == "" {
		return nil, fmt.Errorf("worker_id not found")
	}

	return &ComposerWorkerFieldSet{
		WorkerID: workerId,
	}, nil
}

var _ log.FieldSetReader = &ComposerWorkerFieldSetReader{}

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
	airflowWorkerRunningHostTemplate   = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)
	airflowWorkerMarkingStatusTemplate = regexp.MustCompile(`.*Marking task as\s(?P<state>\S+).\sdag_id=(?P<dagid>\S+),\stask_id=(?P<taskid>\S+),\srun_id=(?P<runid>\S+),\s(map_index=(?P<mapIndex>\d+),\s)?.+`)
)

func (c *ComposerWorkerTaskInstanceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
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

var _ log.FieldSetReader = &ComposerWorkerTaskInstanceFieldSetReader{}
