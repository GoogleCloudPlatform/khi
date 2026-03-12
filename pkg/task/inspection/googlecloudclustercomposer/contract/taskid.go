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

package googlecloudclustercomposer_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GoogleCloudComposerTaskIDPrefix is the prefix for all task ids related to google cloud composer.
var GoogleCloudComposerTaskIDPrefix = "cloud.google.com/composer/"

// ClusterIdentityTaskID is the task id for aliasing the cluster identity.
var ClusterIdentityTaskID = taskid.NewDefaultImplementationID[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](GoogleCloudComposerTaskIDPrefix + "cluster-identity")

// AutocompleteComposerClusterNamesTaskID is the task id for the task that autocompletes GKE cluster names created by Cloud Composer.
var AutocompleteComposerClusterNamesTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.AutocompleteClusterIdentityTaskID.Ref(), "composer")

// ComposerClusterNamePrefixTaskID is the task id for the task that returns the GKE cluster name prefix used by Cloud Composer.
var ComposerClusterNamePrefixTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskRef, "composer")

// AutocompleteComposerEnvironmentNamesTaskID is the task id for the task that autocompletes composer environment names.
var AutocompleteComposerEnvironmentNamesTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-environment-names")

// AutocompleteComposerEnvironmentIdentityTaskID is the task id for the task that autocompletes composer environment identities.
var AutocompleteComposerEnvironmentIdentityTaskID = taskid.NewDefaultImplementationID[*inspectioncore_contract.AutocompleteResult[ComposerEnvironmentIdentity]](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-environment-identities")

// InputComposerEnvironmentNameTaskID is the task id for the task that inputs composer environment name.
var InputComposerEnvironmentNameTaskID taskid.TaskImplementationID[string] = taskid.NewDefaultImplementationID[string](GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name")

// AutocompleteComposerComponentsTaskID is the task id for autocompleting component names from Cloud Monitoring.
var AutocompleteComposerComponentsTaskID = taskid.NewDefaultImplementationID[*inspectioncore_contract.AutocompleteResult[string]](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-components")

// InputComposerComponentsTaskID is the task id for selecting target Composer components.
var InputComposerComponentsTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GoogleCloudComposerTaskIDPrefix + "input/composer/components")

// ComposerLogsQueryTaskID is the task id for the task that queries Logs from Cloud Logging.
var ComposerLogsQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "query-composer-logs")

// ComposerLogsFieldSetReadTaskID is the task id for the task that reads fieldsets from composer logs.
var ComposerLogsFieldSetReadTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "fieldsetread")

// AirflowWorkerLogFilterTaskID is the task id for filtering Airflow worker logs.
var AirflowWorkerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "filter-worker")

// AirflowSchedulerLogFilterTaskID is the task id for filtering Airflow scheduler logs.
var AirflowSchedulerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "filter-scheduler")

// AirflowDagProcessorManagerLogFilterTaskID is the task id for filtering Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "filter-dag-processor-manager")

// AirflowOtherLogFilterTaskID is the task id for filtering other Airflow logs.
var AirflowOtherLogFilterTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "filter-other")

// ComposerLogsTailTaskID is the task id for unifying composer logs feature.
var ComposerLogsTailTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "tail-composer-logs")

// AirflowSchedulerLogGrouperTaskID is the task id for the task that groups Airflow scheduler logs.
var AirflowSchedulerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GoogleCloudComposerTaskIDPrefix + "grouper-scheduler")

// AirflowSchedulerLogIngesterTaskID is the task id for the task that ingests Airflow scheduler logs.
var AirflowSchedulerLogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "ingester-scheduler")

// AirflowSchedulerLogToTimelineMapperTaskID is the task id for the task that maps Airflow scheduler logs to timeline events.
var AirflowSchedulerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "mapper-scheduler")

// AirflowDagProcessorManagerLogSorterTaskID is the task id for the task that sorts Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogSorterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "sorter-dag-processor-manager")

// AirflowDagProcessorManagerLogGrouperTaskID is the task id for the task that groups Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GoogleCloudComposerTaskIDPrefix + "grouper-dag-processor-manager")

// AirflowDagProcessorManagerLogIngesterTaskID is the task id for the task that ingests Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "ingester-dag-processor-manager")

// AirflowDagProcessorManagerLogToTimelineMapperTaskID is the task id for the task that maps Airflow DAG processor manager logs to timeline events.
var AirflowDagProcessorManagerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "mapper-dag-processor-manager")

// AirflowWorkerLogGrouperTaskID is the task id for the task that groups Airflow worker logs.
var AirflowWorkerLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GoogleCloudComposerTaskIDPrefix + "grouper-worker")

// AirflowWorkerLogIngesterTaskID is the task id for the task that ingests Airflow worker logs.
var AirflowWorkerLogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "ingester-worker")

// AirflowWorkerLogToTimelineMapperTaskID is the task id for the task that maps Airflow worker logs to timeline events.
var AirflowWorkerLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "mapper-worker")

// AirflowOtherLogGrouperTaskID is the task id for the task that groups other Airflow logs.
var AirflowOtherLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GoogleCloudComposerTaskIDPrefix + "grouper-other")

// AirflowOtherLogIngesterTaskID is the task id for the task that ingests other Airflow logs.
var AirflowOtherLogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "ingester-other")

// AirflowOtherLogToTimelineMapperTaskID is the task id for the task that maps other Airflow logs to timeline events.
var AirflowOtherLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "mapper-other")

// ComposerEnvironmentListFetcherTaskID is the task id for injecting ComposerEnvironmentListFetcher instance.
var ComposerEnvironmentListFetcherTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentListFetcher](GoogleCloudComposerTaskIDPrefix + "composer-environment-list-fetcher")

// ComposerEnvironmentClusterFinderTaskID is the task id for injecting ComposerEnvironmentClusterFinder instance.
var ComposerEnvironmentClusterFinderTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentClusterFinder](GoogleCloudComposerTaskIDPrefix + "composer-environment-cluster-finder")

// AutocompleteLocationForComposerEnvironmentTaskID is the task id for the task that autocompletes GKE cluster location from Composer environments.
var AutocompleteLocationForComposerEnvironmentTaskID = taskid.NewImplementationID(googlecloudcommon_contract.AutocompleteLocationTaskID.Ref(), "composer")
