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

package composerairflow_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
)

// componentFilterTask creates a log filter task that filters logs for a specific Composer component.
func componentFilterTask(taskID taskid.TaskImplementationID[[]*log.Log], source taskid.TaskReference[[]*log.Log], componentName string) coretask.DefinedTask[[]*log.Log] {
	return inspectiontaskbase.DefineLogFilterTask(
		taskID,
		source,
		func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
			return func(ctx context.Context, l *log.Log) bool {
				component, err := composerairflow.ExtractComposerComponent(l.NodeReader)
				if err != nil {
					return false
				}
				return component == componentName
			}
		},
	)
}

// airflowWorkerLogFilterTask filters logs for the Airflow worker component.
var airflowWorkerLogFilterTask = componentFilterTask(composerairflow.AirflowWorkerLogFilterTaskID, composerairflow.ComposerLogsQueryTaskID.Ref(), "airflow-worker")

// airflowSchedulerLogFilterTask filters logs for the Airflow scheduler component.
var airflowSchedulerLogFilterTask = componentFilterTask(composerairflow.AirflowSchedulerLogFilterTaskID, composerairflow.ComposerLogsQueryTaskID.Ref(), "airflow-scheduler")

// airflowDagProcessorManagerLogFilterTask filters logs for the Airflow DAG processor manager component.
var airflowDagProcessorManagerLogFilterTask = componentFilterTask(composerairflow.AirflowDagProcessorManagerLogFilterTaskID, composerairflow.ComposerLogsQueryTaskID.Ref(), "dag-processor-manager")

// airflowOtherLogFilterTask filters logs for Airflow components without a dedicated pipeline.
var airflowOtherLogFilterTask = inspectiontaskbase.DefineLogFilterTask(
	composerairflow.AirflowOtherLogFilterTaskID,
	composerairflow.ComposerLogsQueryTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			component, err := composerairflow.ExtractComposerComponent(l.NodeReader)
			if err != nil {
				return false
			}
			// If it's none of the specific components we support parsing, it goes to "Other"
			return component != "airflow-worker" && component != "airflow-scheduler" && component != "dag-processor-manager"
		}
	},
)
