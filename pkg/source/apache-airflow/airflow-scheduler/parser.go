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

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	airflow "github.com/GoogleCloudPlatform/khi/pkg/source/apache-airflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// Regex templates to parse Airflow log format
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

// Parse airflow-scheduler logs and make them into TaskInstances.
// This parser will detect these lifecycles;
// - scheduled
// - queued
// - success
// - failed
type AirflowSchedulerParser struct {
	queryTaskId   taskid.TaskReference[[]*log.LogEntity]
	targetLogType enum.LogType
}

func NewAirflowSchedulerParser(queryTaskId taskid.TaskReference[[]*log.LogEntity], targetLogType enum.LogType) *AirflowSchedulerParser {
	return &AirflowSchedulerParser{queryTaskId: queryTaskId, targetLogType: targetLogType}
}

// TargetLogType implements parser.Parser.
func (t *AirflowSchedulerParser) TargetLogType() enum.LogType {
	return t.targetLogType
}

var _ parser.Parser = &AirflowSchedulerParser{}

func (*AirflowSchedulerParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*AirflowSchedulerParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

func (*AirflowSchedulerParser) Description() string {
	return `Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.`
}

func (*AirflowSchedulerParser) GetParserName() string {
	return "Airflow Scheduler"
}

func (a *AirflowSchedulerParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return a.queryTaskId
}

func (t *AirflowSchedulerParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {

	ti, err := t.parseInternal(l)
	if err != nil {
		return err
	}
	if ti == nil { // not found
		return nil
	}

	resourcePath := resourcepath.AirflowTaskInstance(ti)
	verb, state := airflow.TiStatusToVerb(ti)
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       verb,
		State:      state,
		Requestor:  "airflow-scheduler",
		ChangeTime: l.Timestamp(),
		Partial:    false,
		Body:       ti.ToYaml(),
	})

	summary, _ := l.MainMessage()
	cs.RecordLogSummary(summary)
	cs.RecordEvent(resourcePath)

	return nil
}

// parseInternal generates AirflowTaskInstance from the logEntity.
// If the log does not contain information about ti, parseInternal throw non-nil error.
func (t *AirflowSchedulerParser) parseInternal(l *log.LogEntity) (*model.AirflowTaskInstance, error) {

	// TODO since all templates can generate same parametors(dagid,taskid,runid,state,mapIndex), I don't create `airflowParserFn`s for each template.
	// TODO create a generic airflowParserFn which generate ti from a simple template.
	template := []*regexp.Regexp{
		airflowTiTemplate,
		airflowSchedulerReceivedEventTemplate,
		airflowSchedulerTaskFinishedTemplate,
	}

	textPayload, err := l.GetString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe this is not airflow log. please confirm the log. ID: %s", l.ID())
	}

	// iterates through a list of regular expressions to match the log entity against.
	for _, re := range template {

		// If the log entity matches one of the regular expressions,
		// the function extracts the following information from the log message
		matches := re.FindStringSubmatch(textPayload)
		if matches == nil {
			continue
		}
		dagid := matches[re.SubexpIndex("dagid")]
		taskid := matches[re.SubexpIndex("taskid")]
		runid := matches[re.SubexpIndex("runid")]
		stateStr := matches[re.SubexpIndex("state")] // Renamed original string variable
		mapIndex := "-1"                             // optional, applied for only Dynamic DAG.
		if matches[re.SubexpIndex("mapIndex")] != "" {
			mapIndex = matches[re.SubexpIndex("mapIndex")]
		}
		state, err := airflow.StringToTiState(stateStr)
		if err != nil {
			fmt.Printf("Warning: Could not convert Airflow state '%s' to Tistate: %v. Skipping log entry.\n", stateStr, err)
			continue
		}
		return model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, "", state), nil
	}

	matches := airflowSchedulerZombieDetectedTemplate.FindStringSubmatch(textPayload)
	if matches == nil {
		// this log entity is not for TaskInstance lifecycle.
		return nil, nil
	}
	dagid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("dagid")]
	taskid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("taskid")]
	runid := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("runid")]
	state := model.TASKINSTANCE_ZOMBIE
	host := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("host")]
	mapIndex := "-1"
	if i := matches[airflowSchedulerZombieDetectedTemplate.SubexpIndex("mapIndex")]; i != "" {
		mapIndex = i
	}
	return model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state), nil
}
