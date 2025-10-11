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

package googlecloudclustergkeonazure_impl

import (
	"context"
	"fmt"
	"log/slog"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergkeonazure_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteGKEOnAzureClusterNamesTask is a task that provides a list of GKE on Azure cluster names for autocompletion.
var AutocompleteGKEOnAzureClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudclustergkeonazure_contract.AutocompleteGKEOnAzureClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudclustergkeonazure_contract.ClusterListFetcherTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	if projectID != "" && projectID == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID != "" {
		clusterListFetcher := coretask.GetTaskResult(ctx, googlecloudclustergkeonazure_contract.ClusterListFetcherTaskID.Ref())
		clusterNames, err := clusterListFetcher.GetClusterNames(ctx, projectID)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the cluster names in the project %s\n%s", projectID, err))
			return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: projectID,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error:        fmt.Sprintf("Failed to get the list from API:%s", err.Error()),
				},
			}, nil
		}
		return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: projectID,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: clusterNames,
				Error:        "",
			},
		}, nil
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: projectID,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error:        "Project ID is empty",
		},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustergkeonazure_contract.InspectionTypeId))
