// Copyright 2024 Google LLC
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

package composer_task

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
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var (
	// \t<TaskInstance: $DAGID.$TASKID $RUNID map_index=$MAPINDEX [scheduled]>
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1179
	airflowTiTemplate = regexp.MustCompile(`\s<TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]>`)

	// Received executor event with state queued for task instance TaskInstanceKey(dag_id='khi_dag', task_id='add_one', run_id='scheduled__2023-11-30T05:00:00+00:00', try_number=1, map_index=0)
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L685
	airflowSchedulerReceivedEventTemplate = regexp.MustCompile(`Received executor event with state (?P<state>.+) for task instance TaskInstanceKey\(dag_id='(?P<dagid>.+)', task_id='(?P<taskid>.+)', run_id='(?P<runid>.+)',.*map_index=(?P<mapIndex>\d+)\)`)

	// TaskInstance Finished: dag_id=DAGID, task_id=TASKID, run_id=RUNID, map_index=MAPINDEX, ..., state=STATE ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L715
	airflowSchedulerTaskFinishedTemplate = regexp.MustCompile(`TaskInstance Finished:\s+dag_id=(?P<dagid>\S+),\s+task_id=(?P<taskid>\S+),\s+run_id=(?P<runid>\S+),\s+map_index=(?P<mapIndex>\S+),\s+.*state=(?P<state>\S+), executor_state.+`)

	// Detected zombie job: {'full_filepath': '...', 'processor_subdir': '...', 'msg': "{'DAG Id': 'DAG_ID', 'Task Id': 'TASK_ID', 'Run Id': 'RUN_ID', 'Hostname': 'WORKER', ...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/jobs/scheduler_job_runner.py#L1746C55-L1746C62
	airflowSchedulerZombieDetectedTemplate = regexp.MustCompile(`'DAG Id':\s*'(?P<dagid>[^']+)',\s*'Task Id':\s*'(?P<taskid>[^']+)',\s*'Run Id':\s*'(?P<runid>[^']+)',\s*('Map Index':\s*'(?P<mapIndex>[^']+)',\s*)?'Hostname':\s*'(?P<host>[^']+)'`)
)

// convert Taskinstance status to (enum.RevisionVerb, enum.RevisionState)
func tiStatusToVerb(ti *model.AirflowTaskInstance) (enum.RevisionVerb, enum.RevisionState) {
	switch ti.Status() {
	case model.TASKINSTANCE_SCHEDULED:
		return enum.RevisionVerbComposerTaskInstanceScheduled, enum.RevisionStateComposerTiScheduled
	case model.TASKINSTANCE_QUEUED:
		return enum.RevisionVerbComposerTaskInstanceQueued, enum.RevisionStateComposerTiQueued
	case model.TASKINSTANCE_RUNNING:
		return enum.RevisionVerbComposerTaskInstanceRunning, enum.RevisionStateComposerTiRunning
	case model.TASKINSTANCE_SUCCESS:
		return enum.RevisionVerbComposerTaskInstanceSuccess, enum.RevisionStateComposerTiSuccess
	case model.TASKINSTANCE_FAILED:
		return enum.RevisionVerbComposerTaskInstanceFailed, enum.RevisionStateComposerTiFailed
	case model.TASKINSTANCE_DEFERRED:
		return enum.RevisionVerbComposerTaskInstanceDeferred, enum.RevisionStateComposerTiDeferred
	case model.TASKINSTANCE_UP_FOR_RETRY:
		return enum.RevisionVerbComposerTaskInstanceUpForRetry, enum.RevisionStateComposerTiUpForRetry
	case model.TASKINSTANCE_UP_FOR_RESCHEDULE:
		return enum.RevisionVerbComposerTaskInstanceUpForReschedule, enum.RevisionStateComposerTiUpForReschedule
	case model.TASKINSTANCE_REMOVED:
		return enum.RevisionVerbComposerTaskInstanceRemoved, enum.RevisionStateComposerTiRemoved
	case model.TASKINSTANCE_UPSTREAM_FAILED:
		return enum.RevisionVerbComposerTaskInstanceUpstreamFailed, enum.RevisionStateComposerTiUpstreamFailed
	case model.TASKINSTANCE_ZOMBIE:
		return enum.RevisionVerbComposerTaskInstanceZombie, enum.RevisionStateComposerTiZombie
	default:
		return enum.RevisionVerbComposerTaskInstanceUnimplemented, enum.RevisionStateConditionUnknown
	}
}

var AirflowSchedulerLogParseJob = parser.NewParserTaskFromParser(composer_taskid.AirflowSchedulerLogParserTaskID, &AirflowSchedulerParser{}, false, []string{composer_inspection_type.InspectionTypeId})

// Parse airflow-scheduler logs and make them into TaskInstances.
// This parser will detect these lifecycles;
// - scheduled
// - queued
// - success
// - failed
type AirflowSchedulerParser struct {
}

// TargetLogType implements parser.Parser.
func (t *AirflowSchedulerParser) TargetLogType() enum.LogType {
	return enum.LogTypeComposerEnvironment
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
	return "(Alpha) Composer / Airflow Scheduler"
}

func (*AirflowSchedulerParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return composer_taskid.ComposerSchedulerLogQueryTaskID.GetTaskReference()
}

func (t *AirflowSchedulerParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {

	ti, err := t.parseInternal(l)
	if err != nil {
		// return err
		return nil
	}

	resourcePath := resourcepath.ComposerTaskInstance(ti)
	verb, state := tiStatusToVerb(ti)
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       verb,
		State:      state,
		Requestor:  "airflow-scheduler",
		ChangeTime: l.Timestamp(),
		Partial:    false,
		Body:       ti.ToYaml(),
	})

	summary, err := l.MainMessage()
	if err == nil {
		cs.RecordLogSummary(summary)
	}
	return nil
}

// parseInternal generates AirflowTaskInstance from the logEntity.
// If the log does not contain information about ti, parseInternal throw non-nil error.
func (t *AirflowSchedulerParser) parseInternal(l *log.LogEntity) (*model.AirflowTaskInstance, error) {

	// TODO since all templates can generate same parametors(dagid,taskid,runid,state,mapIndex), I don't create `airflowParserFn`s for each template.
	// TODO create a generic airflowParserFn which generate ti from a simple template.
	// TODO b/342036438
	template := []*regexp.Regexp{
		airflowTiTemplate,
		airflowSchedulerReceivedEventTemplate,
		airflowSchedulerTaskFinishedTemplate,
	}

	textPayload, err := l.GetString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe this is not composer log. please confirm the log")
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
		state := matches[re.SubexpIndex("state")]
		mapIndex := "-1" // optional, applied for only Dynamic DAG.
		if matches[re.SubexpIndex("mapIndex")] != "" {
			mapIndex = matches[re.SubexpIndex("mapIndex")]
		}
		return model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, "", state), nil
	}

	matches := airflowSchedulerZombieDetectedTemplate.FindStringSubmatch(textPayload)
	if matches == nil {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
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

var (
	// Running <TaskInstance: DAG_ID.TASK_ID RUN_ID [STATE]> on host WORKER
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/cli/commands/task_command.py#L416
	// airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\S+)\.(?P<taskid>\S+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)
	airflowWorkerRunningHostTemplate = regexp.MustCompile(`Running <TaskInstance:\s(?P<dagid>\w+)\.(?P<taskid>[\w.-]+)\s(?P<runid>\S+)\s(?:map_index=(?P<mapIndex>\d+)\s)?\[(?P<state>\w+)\]> on host (?P<host>.+)`)

	// Marking task as STATE. dag_id=DAG_ID, task_id=TASK_ID, map_index=MAP_INDEX, execution_date=..., start_date=..., end_date=...
	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1396
	airflowWorkerMarkingStatusTemplate = regexp.MustCompile(`.*Marking task as\s(?P<state>\S+).\sdag_id=(?P<dagid>\S+),\stask_id=(?P<taskid>\S+),\s(map_index=(?P<mapIndex>\d+),\s)?.+`)
)

var AirflowWorkerLogParseJob = parser.NewParserTaskFromParser(composer_taskid.AirflowWorkerLogParserTaskID, &AirflowWorkerParser{}, false, []string{composer_inspection_type.InspectionTypeId})

// Parse airflow-scheduler logs and make them into TaskInstances.
// This parser will detect these lifecycles;
// - running
var _ parser.Parser = &AirflowWorkerParser{}

type AirflowWorkerParser struct {
}

// TargetLogType implements parser.Parser.
func (a *AirflowWorkerParser) TargetLogType() enum.LogType {
	return enum.LogTypeComposerEnvironment
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
	return "(Alpha) Cloud Composer / Airflow Worker"
}

// LogTask implements parser.Parser.
func (*AirflowWorkerParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return composer_taskid.ComposerWorkerLogQueryTaskID.GetTaskReference()
}

// Parse implements parser.Parser.
func (*AirflowWorkerParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	parsers := []airflowParserFn{
		&airflowWorkerRunningHostFn{},
		// &airflowWorkerMarkingStatusFn{},
	}

	for _, p := range parsers {
		ti, err := p.fn(l)
		if err != nil {
			continue
		}

		r := resourcepath.ComposerTaskInstance(ti)
		verb, state := tiStatusToVerb(ti)
		cs.RecordRevision(r, &history.StagingResourceRevision{
			Verb:       verb,
			State:      state,
			Requestor:  "airflow-worker", // TODO should change(trigger, scheduler, manual, etc...)
			ChangeTime: l.Timestamp(),
			Partial:    false,
			Body:       ti.ToYaml(),
		})

		worker := model.NewAirflowWorker(ti.Host())
		cs.RecordEvent(resourcepath.ComposerAirflowWorker(worker))
	}

	summary, err := l.MainMessage()
	if err == nil {
		cs.RecordLogSummary(summary)
	}

	return nil
}

type airflowWorkerRunningHostFn struct{}

var _ airflowParserFn = (*airflowWorkerRunningHostFn)(nil)

func (fn *airflowWorkerRunningHostFn) fn(inputLog *log.LogEntity) (*model.AirflowTaskInstance, error) {
	textPayload, err := inputLog.GetString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe invalid log. please confirm the log %s", inputLog.ID())
	}

	// if textPayload does not start from "Running ...", return nil error
	// this early return is for parformance(regex is too slow)
	if !strings.HasPrefix(textPayload, "Running ") {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
	}

	var taskInstance *model.AirflowTaskInstance
	matches := airflowWorkerRunningHostTemplate.FindStringSubmatch(textPayload)
	if matches == nil {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
	}
	dagid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("dagid")]
	taskid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("taskid")]
	runid := matches[airflowWorkerRunningHostTemplate.SubexpIndex("runid")]
	host := matches[airflowWorkerRunningHostTemplate.SubexpIndex("host")]
	state := matches[airflowWorkerRunningHostTemplate.SubexpIndex("state")]
	mapIndex := "-1" // optional, applied for only Dynamic DAG.
	if matches[airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex")] != "" {
		mapIndex = matches[airflowWorkerRunningHostTemplate.SubexpIndex("mapIndex")]
	}
	taskInstance = model.NewAirflowTaskInstance(dagid, taskid, runid, mapIndex, host, state)
	return taskInstance, nil
}

type airflowWorkerMarkingStatusFn struct{}

var _ airflowParserFn = (*airflowWorkerMarkingStatusFn)(nil)

func (fn *airflowWorkerMarkingStatusFn) fn(inputLog *log.LogEntity) (*model.AirflowTaskInstance, error) {

	textPayload, err := inputLog.GetString("textPayload")
	if err != nil {
		return nil, fmt.Errorf("textPayload not found. maybe invalid log. please confirm the log %s", inputLog.ID())
	}

	var taskInstance *model.AirflowTaskInstance
	matches := airflowWorkerMarkingStatusTemplate.FindStringSubmatch(textPayload)
	if matches == nil {
		return nil, fmt.Errorf("this log entity is not for TaskInstance lifecycle. abort")
	}

	workerId, err := inputLog.GetString("labels.worker_id")
	if err != nil {
		return nil, fmt.Errorf("worker_id not found. maybe invalid log. please confirm the log %s", inputLog.ID())
	}

	dagid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("dagid")]
	taskid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("taskid")]

	// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/models/taskinstance.py#L1392
	state := strings.ToLower(matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("state")])

	// runid := matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("runid")]
	mapIndex := "-1" // optional, applied for only Dynamic DAG.
	if matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex")] != "" {
		mapIndex = matches[airflowWorkerMarkingStatusTemplate.SubexpIndex("mapIndex")]
	}
	taskInstance = model.NewAirflowTaskInstance(dagid, taskid, "unknown", mapIndex, workerId, state)
	return taskInstance, nil
}

// airflowParserFn is in charge of "Parse a airflow log, and create a TaskInstance object".
// this interface is for internal
type airflowParserFn interface {
	// fn must return non-nil AirflowTaskInstance if the inputLog indicates a task instance.
	// if there are any errors(i.e textPayload not found), please return nil as AirflowTaskInstance.
	fn(inputLog *log.LogEntity) (*model.AirflowTaskInstance, error)
}

var AirflowDagProcessorLogParseJob = parser.NewParserTaskFromParser(composer_taskid.AirflowDagProcessorManagerLogParserTaskID, &AirflowDagProcessorParser{"/home/airflow/gcs/dags/"}, false, []string{composer_inspection_type.InspectionTypeId})

type AirflowDagProcessorParser struct {
	dagFilePath string
}

// TargetLogType implements parser.Parser.
func (a *AirflowDagProcessorParser) TargetLogType() enum.LogType {
	return enum.LogTypeComposerEnvironment
}

var _ parser.Parser = (*AirflowDagProcessorParser)(nil)

func (*AirflowDagProcessorParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*AirflowDagProcessorParser) Description() string {
	return "The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load."
}

func (*AirflowDagProcessorParser) GetParserName() string {
	return "(Alpha) Composer / Airflow DagProcessorManager"
}

// Grouper implements parser.Parser.
func (*AirflowDagProcessorParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

func (*AirflowDagProcessorParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return composer_taskid.ComposerDagProcessorManagerLogQueryTaskID.GetTaskReference()
}

func (a *AirflowDagProcessorParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	textPayload, err := l.GetString("textPayload")
	if err != nil {
		return fmt.Errorf("textPayload not found. maybe invalid log. please confirm the log %s", l.ID())
	}

	dagFileProcessorStats := a.fromLogEntity(textPayload)
	if dagFileProcessorStats == nil {
		return fmt.Errorf("this is not a dag file processor stats log, skip")
	}
	cs.RecordRevision(resourcepath.DagFileProcessorStats(dagFileProcessorStats), &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbComposerTaskInstanceStats,
		State:      enum.RevisionStateConditionTrue,
		Requestor:  "dag-processor-manager",
		ChangeTime: l.Timestamp(),
		Partial:    false,
		Body: fmt.Sprintf("dags: %s\nerrors: %s",
			dagFileProcessorStats.NumberOfDags(), dagFileProcessorStats.NumberOfErrors()),
	})

	var summary string
	if dagFileProcessorStats.Runtime() != "" {
		summary = fmt.Sprintf("dags=%s, errors=%s, runtime=%s",
			dagFileProcessorStats.NumberOfDags(), dagFileProcessorStats.NumberOfErrors(), dagFileProcessorStats.Runtime())
	} else {
		summary = fmt.Sprintf("dags=%s, errors= %s",
			dagFileProcessorStats.NumberOfDags(), dagFileProcessorStats.NumberOfErrors())
	}
	cs.RecordLogSummary(summary)
	return nil
}

// parse DAG Processor Manager's parse result log.
// Sample: /home/airflow/gcs/dags/main.py 40441 4.06s 64 0 6.93s 2024-05-02T05:14:54
func (a *AirflowDagProcessorParser) fromLogEntity(log string) *model.DagFileProcessorStats {

	// devide the string with " ".
	var fragmentation []string
	for _, s := range strings.Split(log, " ") {
		if s != "" {
			fragmentation = append(fragmentation, s)
		}
	}

	validate := func(f []string) bool {

		// according to the source code, the number of output can be 3, 4, 5, 6, 7
		// https://github.com/apache/airflow/blob/2.7.3/airflow/dag_processing/manager.py#L866
		// case 3 = can happen(file_path, num_dags, num_errors)
		// case 4 = can happen(file_path, num_dags, num_errors, pid or runtime)
		// case 5 = it's a major pattern(file_path, num_dags, num_errors, last_runtime, last_run)
		// case 6 = can happen(file_path, num_dags, num_errors, last_runtime, last_run, pid or runtime)
		// case 7 = it's a major pattern(all)
		if len(f) < 2 || len(f) > 7 {
			return false
		}

		if !strings.HasPrefix(f[0], a.dagFilePath) {
			return false
		}
		return true
	}

	if !validate(fragmentation) {
		return nil
	}

	return func(frags []string) *model.DagFileProcessorStats {
		filePath := frags[0]
		var runtime, numberOfDags, numberOfErrors string

		// runtime and last_runtime must contain "s"
		// ref: https://github.com/apache/airflow/blob/2.7.3/airflow/dag_processing/manager.py#L870

		isRuntime := func(s string) bool {
			return strings.Contains(s, "s")
		}

		switch len(frags) { // the length must be between 3~7(inclusive)
		case 3:
			// FILE_PATH DAG ERROR
			numberOfDags, numberOfErrors = frags[1], frags[2]
		case 4:
			guess := frags[1]
			if isRuntime(guess) {
				// FILE_PATH RUNTIME DAG ERROR
				runtime = frags[1]
			}
			numberOfDags, numberOfErrors = frags[2], frags[3]
		case 5:
			guess := frags[2]
			// FILE_PATH PID RUNTIME DAG ERROR
			if isRuntime(guess) {
				runtime, numberOfDags, numberOfErrors = frags[2], frags[3], frags[4]
			} else { // FILE_PATH DAG ERROR LAST_RUNTIME LAST_RUN
				numberOfDags, numberOfErrors = frags[1], frags[2]
			}
		case 6:
			// FILE_PATH RUNTIME DAG ERROR LAST_RUNTIME LAST_RUN
			guess := frags[1]
			if isRuntime(guess) {
				runtime, numberOfDags, numberOfErrors = frags[1], frags[2], frags[3]
				break
			}

			// FILE_PATH PID RUNTIME DAG ERROR LAST_RUNTIME/LAST_RUN
			// or
			// FILE_PATH PID DAG ERROR LAST_RUNTIME LAST_RUN
			guess = frags[2]
			if isRuntime(guess) {
				runtime, numberOfDags, numberOfErrors = frags[2], frags[3], frags[4]
				break
			}
			numberOfDags, numberOfErrors = frags[2], frags[3]

		case 7:
			// FILE_PATH PID RUNTIME DAG ERROR LAST_RUNTIME LAST_RUN
			runtime, numberOfDags, numberOfErrors = frags[2], frags[3], frags[4]
		}

		return model.NewDagFileProcessorStats(
			filePath,
			runtime,
			numberOfDags,
			numberOfErrors,
		)
	}(fragmentation)
}
