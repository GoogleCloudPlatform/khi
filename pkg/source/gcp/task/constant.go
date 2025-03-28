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

package task

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

const GCPPrefix = "cloud.google.com/"

// ClusterNamePrefixTaskID is the task ID for generating the cluster name prefix used in query.
// For GKE, it's just a task to return "" always.
// For Anthos on AWS, it should return "awsClusters/" because the `resource.labels.cluster_name` field would be `awsClusters/<cluster-name>`
// For Anthos on Azure, it will be "azureClusters/"
var ClusterNamePrefixTaskID = taskid.NewTaskReference[string](GCPPrefix + "cluster-name-prefix")

var K8sResourceMergeConfigTaskID = taskid.NewDefaultImplementationID[*k8s.MergeConfigRegistry](GCPPrefix + "merge-config")

var GCPDefaultK8sResourceMergeConfigTask = task.NewTask(K8sResourceMergeConfigTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (*k8s.MergeConfigRegistry, error) {
	return k8s.GenerateDefaultMergeConfig()
})
