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

package k8scommon_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var ClusterIdentityTask = inspectiontaskbase.NewInspectionTask(k8scommon.ClusterIdentityTaskID, []coretask.Dependency{
	gcpcommon.InputProjectIdTaskID.Ref(),
	k8scommon.InputClusterNameTaskID.Ref(),
	gcpcommon.InputLocationsTaskID.Ref(),
	k8scommon.ClusterNamePrefixTaskRef,
}, func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8scommon.GoogleCloudClusterIdentity, error) {
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, k8scommon.InputClusterNameTaskID.Ref())
	location := coretask.GetTaskResult(ctx, gcpcommon.InputLocationsTaskID.Ref())
	prefixPolicy := coretask.GetTaskResult(ctx, k8scommon.ClusterNamePrefixTaskRef)
	return k8scommon.GoogleCloudClusterIdentity{
		ProjectID:    projectID,
		PrefixPolicy: prefixPolicy,
		ClusterName:  clusterName,
		Location:     location,
	}, nil

})
