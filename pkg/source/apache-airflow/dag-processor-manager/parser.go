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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type AirflowDagProcessorParser struct {
	dagFilePath   string
	logTask       taskid.TaskReference[[]*log.Log]
	targetLogType enum.LogType
}

func NewAirflowDagProcessorParser(dagFilePath string, logTask taskid.TaskReference[[]*log.Log], targetLogType enum.LogType) *AirflowDagProcessorParser {
	return &AirflowDagProcessorParser{
		dagFilePath:   dagFilePath,
		logTask:       logTask,
		targetLogType: targetLogType,
	}
}

// TargetLogType implements parser.Parser.
func (a *AirflowDagProcessorParser) TargetLogType() enum.LogType {
	return a.targetLogType
}

var _ parser.Parser = (*AirflowDagProcessorParser)(nil)

func (*AirflowDagProcessorParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*AirflowDagProcessorParser) Description() string {
	return "The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load."
}

func (*AirflowDagProcessorParser) GetParserName() string {
	return "Airflow DagProcessorManager"
}

// Grouper implements parser.Parser.
func (*AirflowDagProcessorParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

func (a *AirflowDagProcessorParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return a.logTask
}

func (a *AirflowDagProcessorParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	commonField, _ := log.GetFieldSet(l, &log.CommonFieldSet{})
	mainMessage, err := log.GetFieldSet(l, &log.MainMessageFieldSet{})
	if err != nil {
		cs.RecordLogSummary(mainMessage.MainMessage)
	}

	dagFileProcessorStats := a.fromLogEntity(mainMessage.MainMessage)
	if dagFileProcessorStats == nil {
		// this is not a dag file processor stats log, skip
		return nil
	}
	cs.RecordRevision(resourcepath.DagFileProcessorStats(dagFileProcessorStats), &history.StagingResourceRevision{
		Verb:       enum.RevisionVerbComposerTaskInstanceStats,
		State:      enum.RevisionStateConditionTrue,
		Requestor:  "dag-processor-manager",
		ChangeTime: commonField.Timestamp,
		Partial:    false,
		Body: fmt.Sprintf("dags: %s\nerrors: %s",
			dagFileProcessorStats.NumberOfDags(), dagFileProcessorStats.NumberOfErrors()),
	})

	// Emphasize "Error" for parsing dag failures
	if dagFileProcessorStats.NumberOfErrors() != "0" {
		cs.RecordLogSeverity(enum.SeverityError)
	}

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

	// TODO add support for `last_num_of_db_queries` (available from 2.10)
	// The current implementation is based on a fixed number of 6 columns, but this will change to 7 fom 2.10.
	// the implementation needs to support both 6 and 7 columns. Considering future updates, it would be advisable to make the number of columns dynamically adjustable.
	// Fortunately DagProcessorManhager also outputs the column headers to the log. This hint can make adjustments.

	// remove a string "DAG_PROCESSOR_MANAGER_LOG:" from the string(Cloud Composer 3 support)
	log = strings.TrimPrefix(log, "DAG_PROCESSOR_MANAGER_LOG:")

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
