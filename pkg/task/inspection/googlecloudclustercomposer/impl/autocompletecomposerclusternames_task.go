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
	"errors"
	"fmt"

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
	googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinderTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {

	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	environment := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	dependencyDigest := fmt.Sprintf("%s-%s", projectID, environment)

	// when the user is inputing these information, abort
	isWIP := projectID == "" || environment == ""
	if isWIP {
		return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
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

	clusterFinder := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinderTaskID.Ref())
	clusterName, err := clusterFinder.GetGKEClusterName(ctx, projectID, environment)
	if err != nil {
		if errors.Is(err, googlecloudclustercomposer_contract.ErrEnvironmentClusterNotFound) {
			return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: dependencyDigest,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
Note: Composer 3 does not run on your GKE. Please remove all Kubernetes/GKE questies from the previous section.`,
				},
			}, nil
		}
		return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: dependencyDigest,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: []string{},
				Error:        "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
			},
		}, nil
	}

	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: dependencyDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: []string{clusterName},
		},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustercomposer_contract.InspectionTypeId))
