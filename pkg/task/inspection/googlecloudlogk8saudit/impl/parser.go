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

package googlecloudlogk8saudit_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var GCPK8sAuditLogCommonFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogk8saudit_contract.GCPK8sAuditLogCommonFieldSetReaderTaskID,
	googlecloudlogk8saudit_contract.GCPK8sAuditLogListLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8saudit_contract.GCPK8sAuditLogFieldSetReader{},
	},
	inspectioncore_contract.InspectionTypeLabel(googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...),
)

var GCPK8sAuditLogParserTailTask = inspectiontaskbase.NewInspectionTask(
	googlecloudlogk8saudit_contract.GCPK8sAuditLogParserTailTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.LogSummaryHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.NonSuccessLogHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.NamespaceRequestHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceRevisionHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.ConditionHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceOwnerReferenceModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.PodPhaseHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.EndpointResourceHistoryModifierTaskID.Ref(),
		commonlogk8sauditv2_contract.ContainerHistoryModifierTaskID.Ref(),

		commonlogk8sauditv2_contract.NodeNameDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceUIDDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.ContainerIDDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.IPLeaseHistoryDiscoveryTaskID.Ref(),
		googlecloudk8scommon_contract.NEGNamesDiscoveryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel("Kubernetes Audit Log(v3)", `Gather kubernetes audit logs and visualize resource modifications.`, enum.LogTypeAudit, 1001, true, googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...), coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
)
