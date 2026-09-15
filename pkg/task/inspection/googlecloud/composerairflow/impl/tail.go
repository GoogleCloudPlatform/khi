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
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"

	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var ComposerLogsTailTask = coretask.NewTailTask(
	composerairflow.ComposerLogsTailTaskID,
	[]coretask.Dependency{
		composerairflow.AirflowWorkerLogToTimelineMapperTaskID.Ref(),
		composerairflow.AirflowSchedulerLogToTimelineMapperTaskID.Ref(),
		composerairflow.AirflowDagProcessorManagerLogToTimelineMapperTaskID.Ref(),
		composerairflow.AirflowOtherLogToTimelineMapperTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel(
		"Managed Service for Apache Airflow Logs",
		"Gather Managed Service for Apache Airflow logs, including airflow-worker, airflow-scheduler, and airflow-dag-processor-manager, to visualize general environment operations on timelines.",
		2500,
		true,
	),
)
