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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var TailTask = inspectiontaskbase.NewInspectionTask(googlecloudlogk8scontrolplane_contract.TailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8scontrolplane_contract.SchedulerHistoryModifierTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.ControllerManagerHistoryModifierTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.OtherHistoryModifierTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel(
		"Kubernetes Control plane component logs",
		"Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs",
		enum.LogTypeControlPlaneComponent,
		9000,
		false,
		googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...,
	),
)

// LogSerializerTask serializes logs to history for history modifiers to associate event or revisions in later tasks.
// No control plane logs are discarded, thus this LogSerializerTask simply receives logs from the ListLogEntriesTask.
var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(googlecloudlogk8scontrolplane_contract.LogSerializerTaskID, googlecloudlogk8scontrolplane_contract.ListLogEntriesTaskID.Ref())
