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
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	airflowdagprocessor "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/impl/airflow-dag-processor-manager"
	airflowscheduler "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/impl/airflow-scheduler"
	airflowworker "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/impl/airflow-worker"
)

// AirflowSchedulerLogParseTask parses Airflow scheduler logs.
var AirflowSchedulerLogParseTask = legacyparser.NewParserTaskFromParser(
	googlecloudclustercomposer_contract.AirflowSchedulerLogParserTaskID,
	airflowscheduler.NewAirflowSchedulerParser(googlecloudclustercomposer_contract.ComposerSchedulerLogQueryTaskID.Ref(), enum.LogTypeComposerEnvironment),
	100000,
	true,
	[]string{googlecloudclustercomposer_contract.InspectionTypeId},
)

// AirflowWorkerLogParseTask parses Airflow worker logs.
var AirflowWorkerLogParseTask = legacyparser.NewParserTaskFromParser(
	googlecloudclustercomposer_contract.AirflowWorkerLogParserTaskID,
	airflowworker.NewAirflowWorkerParser(googlecloudclustercomposer_contract.ComposerWorkerLogQueryTaskID.Ref(), enum.LogTypeComposerEnvironment),
	101000,
	true,
	[]string{googlecloudclustercomposer_contract.InspectionTypeId},
)

// AirflowDagProcessorLogParseTask parses Airflow DAG processor manager logs.
var AirflowDagProcessorLogParseTask = legacyparser.NewParserTaskFromParser(
	googlecloudclustercomposer_contract.AirflowDagProcessorManagerLogParserTaskID,
	airflowdagprocessor.NewAirflowDagProcessorParser("/home/airflow/gcs/dags/", googlecloudclustercomposer_contract.ComposerDagProcessorManagerLogQueryTaskID.Ref(), enum.LogTypeComposerEnvironment),
	102000,
	true,
	[]string{googlecloudclustercomposer_contract.InspectionTypeId},
)
