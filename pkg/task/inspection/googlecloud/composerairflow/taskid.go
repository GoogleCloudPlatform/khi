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

package composerairflow

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// AutocompleteComposerComponentsTaskID is the task id for autocompleting component names from Cloud Monitoring.
var AutocompleteComposerComponentsTaskID = taskid.NewDefaultImplementationID[*inspectioncore.AutocompleteResult[string]](composercluster.GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-components")

// InputComposerComponentsTaskID is the task id for selecting target Composer components.
var InputComposerComponentsTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](composercluster.GoogleCloudComposerTaskIDPrefix + "input/composer/components")

// ComposerLogsQueryTaskID is the task id for the task that queries Logs from Cloud Logging.
var ComposerLogsQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](composercluster.GoogleCloudComposerTaskIDPrefix + "query-composer-logs")

// AirflowWorkerLogFilterTaskID is the task id for filtering Airflow worker logs.
var AirflowWorkerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](composercluster.GoogleCloudComposerTaskIDPrefix + "filter-worker")

// AirflowSchedulerLogFilterTaskID is the task id for filtering Airflow scheduler logs.
var AirflowSchedulerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](composercluster.GoogleCloudComposerTaskIDPrefix + "filter-scheduler")

// AirflowDagProcessorManagerLogFilterTaskID is the task id for filtering Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](composercluster.GoogleCloudComposerTaskIDPrefix + "filter-dag-processor-manager")

// AirflowOtherLogFilterTaskID is the task id for filtering other Airflow logs.
var AirflowOtherLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](composercluster.GoogleCloudComposerTaskIDPrefix + "filter-other")

// ComposerLogsTailTaskID is the task id for unifying composer logs feature.
var ComposerLogsTailTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "tail-composer-logs")

// AirflowSchedulerLogGrouperTaskID is the task id for the task that groups Airflow scheduler logs.
var AirflowSchedulerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](composercluster.GoogleCloudComposerTaskIDPrefix + "grouper-scheduler")

// AirflowSchedulerLogIngesterTaskID is the task id for the task that ingests Airflow scheduler logs.
var AirflowSchedulerLogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "ingester-scheduler")

// AirflowSchedulerLogToTimelineMapperTaskID is the task id for the task that maps Airflow scheduler logs to timeline events.
var AirflowSchedulerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "mapper-scheduler")

// AirflowDagProcessorManagerLogGrouperTaskID is the task id for the task that groups Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](composercluster.GoogleCloudComposerTaskIDPrefix + "grouper-dag-processor-manager")

// AirflowDagProcessorManagerLogIngesterTaskID is the task id for the task that ingests Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "ingester-dag-processor-manager")

// AirflowDagProcessorManagerLogToTimelineMapperTaskID is the task id for the task that maps Airflow DAG processor manager logs to timeline events.
var AirflowDagProcessorManagerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "mapper-dag-processor-manager")

// AirflowWorkerLogGrouperTaskID is the task id for the task that groups Airflow worker logs.
var AirflowWorkerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](composercluster.GoogleCloudComposerTaskIDPrefix + "grouper-worker")

// AirflowWorkerLogIngesterTaskID is the task id for the task that ingests Airflow worker logs.
var AirflowWorkerLogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "ingester-worker")

// AirflowWorkerLogToTimelineMapperTaskID is the task id for the task that maps Airflow worker logs to timeline events.
var AirflowWorkerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "mapper-worker")

// AirflowOtherLogGrouperTaskID is the task id for the task that groups other Airflow logs.
var AirflowOtherLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](composercluster.GoogleCloudComposerTaskIDPrefix + "grouper-other")

// AirflowOtherLogIngesterTaskID is the task id for the task that ingests other Airflow logs.
var AirflowOtherLogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "ingester-other")

// AirflowOtherLogToTimelineMapperTaskID is the task id for the task that maps other Airflow logs to timeline events.
var AirflowOtherLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](composercluster.GoogleCloudComposerTaskIDPrefix + "mapper-other")
