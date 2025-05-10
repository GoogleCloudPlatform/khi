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

package apacheairflow

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	apacheairflow "github.com/GoogleCloudPlatform/khi/pkg/source/apache-airflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var (
	// Running <TaskInstance: DAG_ID.TASK_ID RUN_ID [STATE]> on host WORKER
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/cli/commands/task_command.py#L416
	// airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\S+)\.(?P<taskid>\S+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)
	airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)

	// Marking task as STATE. dag_id=DAG_ID, task_id=TASK_ID, run_id=RUN_ID, map_index=MAP_INDEX, execution_date=..., start_date=..., end_date=...
	// ref: https://github.com/apache/airflow/blob/2.9.3/airflow/models/taskinstance.py#L1201
	airflowWorkerMarkingStatusTemplate = regexp.MustCompile(`.*Marking task as\s(?P<state>\S+).\sdag_id=(?P<dagid>\S+),\stask_id=(?P<taskid>\S+),\srun_id=(?P<runid>\S+),\s(map_index=(?P<mapIndex>\d+),\s)?.+`)
)

// Parse airflow-scheduler logs and make them into TaskInstances.
// This parser will detect these lifecycles;
// - running
var _ parser.Parser = &AirflowWorkerParser{}

type AirflowWorkerParser struct {
	queryTaskId   taskid.TaskReference[[]*log.Log]
	targetLogType enum.LogType
}

func NewAirflowWorkerParser(queryTaskId taskid.TaskReference[[]*log.Log], targetLogType enum.LogType) *AirflowWorkerParser {
	return &AirflowWorkerParser{
		queryTaskId:   queryTaskId,
		targetLogType: targetLogType,
	}
}

// TargetLogType implements parser.Parser.
func (a *AirflowWorkerParser) TargetLogType() enum.LogType {
	return a.targetLogType
}

// Dependencies implements parser.Parser.
func (*AirflowWorkerParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// DependsOnPast implements parser.Parser.
func (*AirflowWorkerParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Description implements parser.Parser.
func (*AirflowWorkerParser) Description() string {
	return `Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.`
}

// GetParserName implements parser.Parser.
func (*AirflowWorkerParser) GetParserName() string {
	return "Airflow Worker"
}

// LogTask implements parser.Parser.
func (a *AirflowWorkerParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return a.queryTaskId
}

// Parse implements parser.Parser.
func (*AirflowWorkerParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	parsers := []airflowParserFn{
		&airflowWorkerRunningHostFn{},
		&airflowWorkerMarkingStatusFn{},
	}

	host, _ := l.ReadString("labels.worker_id")
	worker := model.NewAirflowWorker(host)
	cs.RecordEvent(resourcepath.AirflowWorker(worker))
	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err != nil {
		cs.RecordLogSummary(mainMessage.MainMessage)
	}

	for _, p := range parsers {
		ti, err := p.fn(l)
		if err != nil {
			continue
		}

		r := resourcepath.AirflowTaskInstance(ti)
		verb, state := apacheairflow.TiStatusToVerb(ti)
		cs.RecordRevision(r, &history.StagingResourceRevision{
			Verb:       verb,
			State:      state,
			Requestor:  "airflow-worker",
			ChangeTime: commonField.Timestamp,
			Partial:    false,
			Body:       ti.ToYaml(),
		})
	}

	return nil
}

// This fn publishes a Running state of Ti on airflow-worker
type airflowWorkerRunningHostFn struct{}

var _ airflowParserFn = (*airflowWorkerRunningHostFn)(nil)

func (fn *airflowWorkerRunningHostFn) fn(inputLog *log.Log) (*model.AirflowTaskInstance, error) {
	mainMessage, err := log.GetFieldSet(inputLog, &log.MainMessageFieldSet{})
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe invalid log. please confirm the log %s", inputLog.ID)
	}

	// if textPayload does not start from "Running ...", return nil error
	// this early return is for parformance(regex is too slow)
	if !strings.HasPrefix(mainMessage.MainMessage, "Running ") {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
	}

	var taskInstance *model.AirflowTaskInstance
	matches := airflowWorkerRunningHostTemplate.FindStringSubmatch(mainMessage.MainMessage)
	if matches == nil {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
	}
	dagid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("dagid")]
	taskid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("taskid")]
	runid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("runid")]
	host := matches[airflowWorkerRunningHostTemplate.SubexpIndex("host")]
	stateStr := matches[airflowWorkerRunningHostTemplate.SubexpIndex("state")] // Renamed original string variable
	state, err := apacheairflow.StringToTiState(stateStr)
	if err != nil {
		// Log or handle the error appropriately if the state string is unknown.
		fmt.Printf("Warning: Could not convert Airflow state '%s' to Tistate: %v. Skipping log entry.\n", stateStr, err)
		return nil, err // Return error to skip processing this log entry
	}
	mapIndex := "-1" // optional, applied for only Dynamic DAG.
	if matches[airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex")] != "" {
		mapIndex = matches[airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex")]
	}
	taskInstance = model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state)
	return taskInstance, nil
}

// This fn publish the final state of Ti on airflow-worker
type airflowWorkerMarkingStatusFn struct{}

var _ airflowParserFn = (*airflowWorkerMarkingStatusFn)(nil)

func (fn *airflowWorkerMarkingStatusFn) fn(inputLog *log.Log) (*model.AirflowTaskInstance, error) {
	mainMessage, err := log.GetFieldSet(inputLog, &log.MainMessageFieldSet{})
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe invalid log. please confirm the log %s", inputLog.ID)
	}

	var taskInstance *model.AirflowTaskInstance
	matches := airflowWorkerMarkingStatusTemplate.FindStringSubmatch(mainMessage.MainMessage)
	if matches == nil {
		return nil, fmt.Errorf("this entity is not for TaskInstance lifecycle. abort")
	}

	workerId, err := inputLog.ReadString("labels.worker_id") // TODO remove Cloud Logging Dependency
	if err != nil {
		return nil, fmt.Errorf("worker_id not found. maybe invalid log. please confirm the log %s", inputLog.ID)
	}

	dagid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("dagid")]
	taskid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("taskid")]
	runid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("runid")]

	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1392
	state := strings.ToLower(matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("state")])

	// runid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("runid")]
	mapIndex := "-1" // optional, applied for only Dynamic DAG.
	if matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex")] != "" {
		mapIndex = matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex")]
	}
	// Convert the string state to the required model.Tistate type
	tiState, err := apacheairflow.StringToTiState(state)
	if err != nil {
		// Log or handle the error appropriately if the state string is unknown.
		fmt.Printf("Warning: Could not convert Airflow state '%s' to Tistate: %v. Skipping log entry.\n", state, err)
		return nil, err // Return error to skip processing this log entry
	}
	taskInstance = model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, workerId, tiState)
	return taskInstance, nil
}

// airflowParserFn is in charge of "Parse a airflow log, and create a TaskInstance object".
// this interface is for internal
type airflowParserFn interface {
	// fn must return non-nil AirflowTaskInstance if the inputLog indicates a task instance.
	// if there are any errors(i.e textPayload not found), please return nil as AirflowTaskInstance.
	fn(inputLog *log.Log) (*model.AirflowTaskInstance, error)
}
