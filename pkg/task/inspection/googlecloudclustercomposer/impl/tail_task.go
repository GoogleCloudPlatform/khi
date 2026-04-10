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

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var ComposerLogsTailTask = coretask.NewTask(
	googlecloudclustercomposer_contract.ComposerLogsTailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.AirflowWorkerLogToTimelineMapperTaskID.Ref(),
		googlecloudclustercomposer_contract.AirflowSchedulerLogToTimelineMapperTaskID.Ref(),
		googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogToTimelineMapperTaskID.Ref(),
		googlecloudclustercomposer_contract.AirflowOtherLogToTimelineMapperTaskID.Ref(),
	},
	func(ctx context.Context) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel(
		"Composer Logs",
		"Cloud Composer related logs like airflow-worker, airflow-scheduler, airflow-dag-processor-manager, and others.",
		enum.LogTypeComposerEnvironment,
		101000,
		true,
	),
)
