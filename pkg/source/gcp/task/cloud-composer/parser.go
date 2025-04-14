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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	airflowscheduler "github.com/GoogleCloudPlatform/khi/pkg/source/apache-airflow/airflow-scheduler"
	airflowworker "github.com/GoogleCloudPlatform/khi/pkg/source/apache-airflow/airflow-worker"
	airflowdagprocessor "github.com/GoogleCloudPlatform/khi/pkg/source/apache-airflow/dag-processor-manager"
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
)

var AirflowSchedulerLogParseJob = parser.NewParserTaskFromParser(
	composer_taskid.AirflowSchedulerLogParserTaskID,
	airflowscheduler.NewAirflowSchedulerParser(composer_taskid.ComposerSchedulerLogQueryTaskID.GetTaskReference(), enum.LogTypeComposerEnvironment),
	true,
	[]string{composer_inspection_type.InspectionTypeId},
)
var AirflowWorkerLogParseJob = parser.NewParserTaskFromParser(
	composer_taskid.AirflowWorkerLogParserTaskID,
	airflowworker.NewAirflowWorkerParser(composer_taskid.ComposerWorkerLogQueryTaskID.GetTaskReference(), enum.LogTypeComposerEnvironment),
	true,
	[]string{composer_inspection_type.InspectionTypeId},
)
var AirflowDagProcessorLogParseJob = parser.NewParserTaskFromParser(
	composer_taskid.AirflowDagProcessorManagerLogParserTaskID,
	airflowdagprocessor.NewAirflowDagProcessorParser("/home/airflow/gcs/dags/", composer_taskid.ComposerDagProcessorManagerLogQueryTaskID.GetTaskReference(), enum.LogTypeComposerEnvironment),
	true,
	[]string{composer_inspection_type.InspectionTypeId},
)
