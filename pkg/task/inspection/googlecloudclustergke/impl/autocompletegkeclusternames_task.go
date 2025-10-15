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

package googlecloudclustergke_impl

import (
	"context"
	"fmt"
	"log/slog"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergke_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteGKEClusterNamesTask is a task that provides autocomplete suggestions for GKE cluster names.
var AutocompleteGKEClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudclustergke_contract.AutocompleteGKEClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudclustergke_contract.ClusterListFetcherTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {
	listFetcher := coretask.GetTaskResult(ctx, googlecloudclustergke_contract.ClusterListFetcherTaskID.Ref())

	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	if projectID != "" && projectID == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID != "" {
		clusters, err := listFetcher.GetClusterNames(ctx, projectID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to read cluster names for project", "projectID", projectID, "error", err)
			return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: projectID,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error:        fmt.Sprintf("Failed to list GKE cluster names: %v", err),
				},
			}, nil
		}
		return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: projectID,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: clusters,
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
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustergke_contract.InspectionTypeId))
