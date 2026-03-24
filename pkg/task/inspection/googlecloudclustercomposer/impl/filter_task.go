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

package googlecloudclustercomposer_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

func componentFilterTask(taskID taskid.TaskImplementationID[[]*log.Log], source taskid.TaskReference[[]*log.Log], componentName string) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewLogFilterTask(
		taskID,
		source,
		func(ctx context.Context, l *log.Log) bool {
			fs, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
			if err != nil {
				return false
			}
			return fs.Component == componentName
		},
	)
}

var AirflowWorkerLogFilterTask = componentFilterTask(googlecloudclustercomposer_contract.AirflowWorkerLogFilterTaskID, googlecloudclustercomposer_contract.ComposerLogsFieldSetReadTaskID.Ref(), "airflow-worker")
var AirflowSchedulerLogFilterTask = componentFilterTask(googlecloudclustercomposer_contract.AirflowSchedulerLogFilterTaskID, googlecloudclustercomposer_contract.ComposerLogsFieldSetReadTaskID.Ref(), "airflow-scheduler")
var AirflowDagProcessorManagerLogFilterTask = componentFilterTask(googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogFilterTaskID, googlecloudclustercomposer_contract.ComposerLogsFieldSetReadTaskID.Ref(), "dag-processor-manager")

var AirflowOtherLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudclustercomposer_contract.AirflowOtherLogFilterTaskID,
	googlecloudclustercomposer_contract.ComposerLogsFieldSetReadTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		fs, err := log.GetFieldSet(l, &googlecloudclustercomposer_contract.ComposerFieldSet{})
		if err != nil {
			return false
		}
		// If it's none of the specific components we support parsing, it goes to "Other"
		return fs.Component != "airflow-worker" && fs.Component != "airflow-scheduler" && fs.Component != "dag-processor-manager"
	},
)
