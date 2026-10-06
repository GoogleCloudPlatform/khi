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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/summary"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// clusterIdentityTask creates the cluster identity from form inputs and prefix policy.
var clusterIdentityTask = inspectiontaskbase.DefineInspectionTask(k8scommon.ClusterIdentityTaskID, func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[k8scommon.GoogleCloudClusterIdentity] {
	projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
	clusterName := coretask.Use(b, k8scommon.InputClusterNameTaskID.Ref())
	location := coretask.Use(b, gcpcommon.InputLocationsTaskID.Ref())
	prefixPolicy := coretask.Use(b, k8scommon.ClusterNamePrefixTaskRef)
	return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8scommon.GoogleCloudClusterIdentity, error) {
		pID := projectID.Get(ctx)
		cName := clusterName.Get(ctx)
		loc := location.Get(ctx)
		if taskMode == inspectioncore.TaskModeRun {
			if pID != "" {
				summary.SetCoreLabel(ctx, "projectId", pID)
			}
			if cName != "" {
				summary.SetCoreLabel(ctx, "clusterName", cName)
			}
			if loc != "" {
				summary.SetCoreLabel(ctx, "location", loc)
			}
		}
		return k8scommon.GoogleCloudClusterIdentity{
			ProjectID:    pID,
			PrefixPolicy: prefixPolicy.Get(ctx),
			ClusterName:  cName,
			Location:     loc,
		}, nil
	}
})
