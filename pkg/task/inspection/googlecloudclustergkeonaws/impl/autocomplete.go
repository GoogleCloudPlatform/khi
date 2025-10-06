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

package googlecloudclustergkeonaws_impl

import (
	"context"
	"fmt"
	"log/slog"

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergkeonaws_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteGKEOnAWSClusterNames is a task that provides a list of GKE on AWS cluster names for autocompletion.
var AutocompleteGKEOnAWSClusterNames = inspectiontaskbase.NewCachedTask(googlecloudclustergkeonaws_contract.AutocompleteGKEOnAWSClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {
	client, err := googlecloudapi.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{}, err
	}

	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	if projectID != "" && projectID == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID != "" {
		clusterNames, err := client.GetAnthosAWSClusterNames(ctx, projectID)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the cluster names in the project %s\n%s", projectID, err))
			return inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: projectID,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error:        "Failed to get the list from API",
				},
			}, nil
		}
		return inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: projectID,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: clusterNames,
				Error:        "",
			},
		}, nil
	}
	return inspectiontaskbase.CachableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: projectID,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error:        "Project ID is empty",
		},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustergkeonaws_contract.InspectionTypeId))
