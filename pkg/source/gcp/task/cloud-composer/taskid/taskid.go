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

package composer_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

const ComposerQueryPrefix = gcp_task.GCPPrefix + "query/composer/"

// AutocompleteClusterNamesTaskID is an task ID to get list of cluster names on the suggestion popup.
var AutocompleteClusterNamesTaskID taskid.TaskImplementationID[*gcp_task.AutocompleteClusterNameList] = taskid.NewImplementationID(gcp_task.AutocompleteClusterNamesTaskID, "composer")
var AutocompleteComposerEnvironmentNamesTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](gcp_task.GCPPrefix + "autocomplete/composer-environment-names")
var InputComposerEnvironmentTaskID taskid.TaskImplementationID[string] = taskid.NewDefaultImplementationID[string](gcp_task.GCPPrefix + "input/composer/environment_name")

var ComposerSchedulerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](ComposerQueryPrefix + "scheduler")
var ComposerDagProcessorManagerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](ComposerQueryPrefix + "dag-processor-manager")
var ComposerMonitoringLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](ComposerQueryPrefix + "monitoring")
var ComposerWorkerLogQueryTaskID taskid.TaskImplementationID[[]*log.Log] = taskid.NewDefaultImplementationID[[]*log.Log](ComposerQueryPrefix + "worker")

var AirflowSchedulerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](gcp_task.GCPPrefix + "composer/scheduler")
var AirflowDagProcessorManagerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](gcp_task.GCPPrefix + "composer/worker")
var AirflowWorkerLogParserTaskID taskid.TaskImplementationID[struct{}] = taskid.NewDefaultImplementationID[struct{}](gcp_task.GCPPrefix + "composer/dagprocessor")
