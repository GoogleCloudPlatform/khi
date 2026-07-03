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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

var OSSK8sAuditLogFieldExtractorTask = inspectiontaskbase.NewFieldSetReadTask(
	ossclusterk8s_contract.OSSK8sAuditLogProviderTaskID,
	ossclusterk8s_contract.NonEventAuditLogFilterTaskID.Ref(),
	[]log.FieldSetReader{(&ossclusterk8s_contract.OSSK8sAuditLogFieldSetReader{})},
)

var OSSK8sAuditLogParserTailTask = inspectiontaskbase.NewInspectionTask(
	ossclusterk8s_contract.OSSK8sAuditLogParserTailTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8saudit_contract.NonSuccessLogLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.NamespaceRequestLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.ConditionLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.ResourceOwnerReferenceTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.PodPhaseLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.EndpointResourceLogToTimelineMapperTaskID.Ref(),
		commonlogk8saudit_contract.ContainerLogToTimelineMapperTaskID.Ref(),

		commonlogk8saudit_contract.NodeNameDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.ResourceUIDDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.ContainerIDDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.IPLeaseHistoryDiscoveryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel("Kubernetes Audit Logs", `Gather Kubernetes audit logs to visualize resource modifications and API call histories on associated timelines.`, 1001, true), coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
)
