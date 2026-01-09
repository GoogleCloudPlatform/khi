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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// GoogleCloudComposerTaskIDPrefix is the prefix for all task ids related to google cloud composer.
var GoogleCloudComposerTaskIDPrefix = "cloud.google.com/composer/"

// AutocompleteComposerClusterNamesTaskID is the task id for the task that autocompletes GKE cluster names created by Cloud Composer.
var AutocompleteComposerClusterNamesTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID.Ref(), "composer")

// ComposerClusterNamePrefixTaskID is the task id for the task that returns the GKE cluster name prefix used by Cloud Composer.
var ComposerClusterNamePrefixTaskID = taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskRef, "composer")

// AutocompleteComposerEnvironmentNamesTaskID is the task id for the task that autocompletes composer environment names.
var AutocompleteComposerEnvironmentNamesTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-environment-names")

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

// AirflowSchedulerLogParserTaskID is the task id for the task that parses Airflow scheduler logs.
var AirflowSchedulerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "composer/scheduler")

// AirflowDagProcessorManagerLogParserTaskID is the task id for the task that parses Airflow DAG processor manager logs.
var AirflowDagProcessorManagerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "composer/worker")

// AirflowWorkerLogParserTaskID is the task id for the task that parses Airflow worker logs.
var AirflowWorkerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](GoogleCloudComposerTaskIDPrefix + "composer/dagprocessor")

// ComposerEnvironmentListFetcherTaskID is the task id for injecting ComposerEnvironmentListFetcher instance.
var ComposerEnvironmentListFetcherTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentListFetcher](GoogleCloudComposerTaskIDPrefix + "composer-environment-list-fetcher")

// ComposerEnvironmentClusterFinderTaskID is the task id for injecting ComposerEnvironmentClusterFinder instance.
var ComposerEnvironmentClusterFinderTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentClusterFinder](GoogleCloudComposerTaskIDPrefix + "composer-environment-cluster-finder")
