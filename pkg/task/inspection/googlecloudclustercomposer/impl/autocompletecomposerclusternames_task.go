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
	"context"
	"fmt"

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteComposerClusterNamesTask is an implementation for googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID
// the task returns GKE cluster name where the provided Composer environment is running.
var AutocompleteComposerClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudclustercomposer_contract.AutocompleteComposerClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {

	client, err := googlecloudapi.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{}, err
	}

	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	environment := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	dependencyDigest := fmt.Sprintf("%s-%s", projectID, environment)

	// when the user is inputing these information, abort
	isWIP := projectID == "" || environment == ""
	if isWIP {
		return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: dependencyDigest,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
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
		return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: dependencyDigest,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: []string{},
				Error:        "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
			},
		}, nil
	}

	// pickup Cluster if cluster.ResourceLabels contains `goog-composer-environment={environment}`
	// = the gke cluster where the composer is running
	for _, cluster := range clusters {
		if cluster.ResourceLabels["goog-composer-environment"] == environment {
			return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: dependencyDigest,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{cluster.Name},
				},
			}, nil
		}
	}

	return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: dependencyDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
			Note: Composer 3 does not run on your GKE. Please remove all Kubernetes/GKE questies from the previous section.`,
		},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustercomposer_contract.InspectionTypeId))
