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

// ComposerSchedulerLogQueryTaskID is the task id for the task that queries scheduler logs from Cloud Logging.
var ComposerSchedulerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "query-scheduler")

// ComposerDagProcessorManagerLogQueryTaskID is the task id for the task that queries DAG processor manager logs from Cloud Logging.
var ComposerDagProcessorManagerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "query-dag-processor-manager")

// ComposerMonitoringLogQueryTaskID is the task id for the task that queries monitoring logs from Cloud Logging.
var ComposerMonitoringLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "query-monitoring")

// ComposerWorkerLogQueryTaskID is the task id for the task that queries worker logs from Cloud Logging.
var ComposerWorkerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "query-worker")

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

// ComposerEnvironmentListFetcherTaskID is the task id for injecting ComposerEnvironmentListFetcher instance.
var ComposerEnvironmentListFetcherTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentListFetcher](GoogleCloudComposerTaskIDPrefix + "composer-environment-list-fetcher")

// ComposerEnvironmentClusterFinderTaskID is the task id for injecting ComposerEnvironmentClusterFinder instance.
var ComposerEnvironmentClusterFinderTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentClusterFinder](GoogleCloudComposerTaskIDPrefix + "composer-environment-cluster-finder")

// AutocompleteLocationForComposerEnvironmentTaskID is the task id for the task that autocompletes GKE cluster location from Composer environments.
var AutocompleteLocationForComposerEnvironmentTaskID = taskid.NewImplementationID(googlecloudcommon_contract.AutocompleteLocationTaskID.Ref(), "composer")

// ComposerSchedulerFieldSetReadTaskID is the task id for the task that reads fieldset from scheduler logs.
var ComposerSchedulerFieldSetReadTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "fieldsetread-scheduler")

// ComposerDagProcessorManagerFieldSetReadTaskID is the task id for the task that reads fieldset from DAG processor manager logs.
var ComposerDagProcessorManagerFieldSetReadTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "fieldsetread-dag-processor-manager")

// ComposerMonitoringFieldSetReadTaskID is the task id for the task that reads fieldset from monitoring logs.
var ComposerMonitoringFieldSetReadTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "fieldsetread-monitoring")

// ComposerWorkerFieldSetReadTaskID is the task id for the task that reads fieldset from worker logs.
var ComposerWorkerFieldSetReadTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](GoogleCloudComposerTaskIDPrefix + "fieldsetread-worker")
