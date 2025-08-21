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

package googlecloudclustergdcvmware_impl

import (
	"context"
	"fmt"
	"log/slog"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
	googlecloudclustergdcvmware_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcvmware/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// AutocompleteGDCVForVMWareClusterNamesTask is a task that provides autocomplete suggestions for GDCV for VMWare cluster names.
var AutocompleteGDCVForVMWareClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudclustergdcvmware_contract.AutocompleteGDCVForVMWareClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{}, err
	}

	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	if projectID != "" && projectID == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID != "" {
		clusterNames, err := client.GetAnthosOnVMWareClusterNames(ctx, projectID)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the cluster names in the project %s\n%s", projectID, err))
			return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
				DependencyDigest: projectID,
				Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error:        "Failed to get the list from API",
				},
			}, nil
		}
		return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
			DependencyDigest: projectID,
			Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
				ClusterNames: clusterNames,
				Error:        "",
			},
		}, nil
	}
	return inspectiontaskbase.PreviousTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: projectID,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error:        "Project ID is empty",
		},
	}, nil
}, inspection_contract.InspectionTypeLabel(googlecloudclustergdcvmware_contract.InspectionTypeId))
