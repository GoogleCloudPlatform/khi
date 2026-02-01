// Copyright 2026 Google LLC
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

package googlecloudk8scommon_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	taskid "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var ClusterIdentityTask = inspectiontaskbase.NewInspectionTask(googlecloudk8scommon_contract.ClusterIndentityTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudcommon_contract.InputLocationsTaskID.Ref(),
	googlecloudk8scommon_contract.ClusterNamePrefixTaskRef,
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (googlecloudk8scommon_contract.GoogleCloudClusterIdentity, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	location := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputLocationsTaskID.Ref())
	clusterTypePrefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskRef)
	return googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:         projectID,
		ClusterTypePrefix: clusterTypePrefix,
		ClusterName:       clusterName,
		Location:          location,
	}, nil
})
