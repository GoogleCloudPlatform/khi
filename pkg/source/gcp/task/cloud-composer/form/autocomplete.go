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

package composer_form

import (
	"context"
	"fmt"

	inspection_cached_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/cached_task"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// This is an implementation for gcp_task.AutocompleteClusterNamesTaskID
// the task returns GKE cluster name where the provided Composer environment is running
var AutocompleteClusterNames = inspection_cached_task.NewCachedTask(composer_taskid.AutocompleteClusterNamesTaskID, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID.Ref(),
	composer_taskid.InputComposerEnvironmentTaskID.Ref(),
}, func(ctx context.Context, prevValue inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]) (inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList], error) {

	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{}, err
	}

	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	environment := task.GetTaskResult(ctx, composer_taskid.InputComposerEnvironmentTaskID.Ref())
	dependencyDigest := fmt.Sprintf("%s-%s", projectID, environment)

	// when the user is inputing these information, abort
	isWIP := projectID == "" || environment == ""
	if isWIP {
		return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
			DependencyDigest: dependencyDigest,
			Value: &gcp_task.AutocompleteClusterNameList{
				ClusterNames: []string{},
				Error:        "Project ID or Composer environment name is empty",
			},
		}, nil
	}

	if environment != "" && dependencyDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	// fetch all GKE clusters in the project
	clusters, err := client.GetClusters(ctx, projectID)
	if err != nil {
		return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
			DependencyDigest: dependencyDigest,
			Value: &gcp_task.AutocompleteClusterNameList{
				ClusterNames: []string{},
				Error:        "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
			},
		}, nil
	}

	// pickup Cluster if cluster.ResourceLabels contains `goog-composer-environment={environment}`
	// = the gke cluster where the composer is running
	for _, cluster := range clusters {
		if cluster.ResourceLabels["goog-composer-environment"] == environment {
			return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
				DependencyDigest: dependencyDigest,
				Value: &gcp_task.AutocompleteClusterNameList{
					ClusterNames: []string{cluster.Name},
				},
			}, nil
		}
	}

	return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
		DependencyDigest: dependencyDigest,
		Value: &gcp_task.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
			Note: Composer 3 does not run on your GKE. Please remove all Kubernetes/GKE questies from the previous section.`,
		},
	}, nil
}, inspection_task.InspectionTypeLabel(composer_inspection_type.InspectionTypeId))
