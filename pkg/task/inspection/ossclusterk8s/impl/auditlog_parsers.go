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

package ossclusterk8s_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

var OSSK8sAuditLogFieldExtractorTask = inspectiontaskbase.NewFieldSetReadTask(
	ossclusterk8s_contract.OSSK8sAuditLogProviderTaskID,
	ossclusterk8s_contract.NonEventAuditLogFilterTaskID.Ref(),
	[]log.FieldSetReader{(&ossclusterk8s_contract.OSSK8sAuditLogFieldSetReader{})},
	inspectioncore_contract.InspectionTypeLabel(ossclusterk8s_contract.InspectionTypeID),
)

var OSSK8sAuditLogParserTailTask = inspectiontaskbase.NewInspectionTask(
	ossclusterk8s_contract.OSSK8sAuditLogParserTailTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.LogSummaryLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.NonSuccessLogLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.NamespaceRequestLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.ConditionLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceOwnerReferenceTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.PodPhaseLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.EndpointResourceLogToTimelineMapperTaskID.Ref(),
		commonlogk8sauditv2_contract.ContainerLogToTimelineMapperTaskID.Ref(),

		commonlogk8sauditv2_contract.NodeNameDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.ResourceUIDDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.ContainerIDDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.IPLeaseHistoryDiscoveryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel("Kubernetes Audit Log(v3)", `Gather kubernetes audit logs and visualize resource modifications.`, enum.LogTypeAudit, 1001, true, ossclusterk8s_contract.InspectionTypeID), coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
)
