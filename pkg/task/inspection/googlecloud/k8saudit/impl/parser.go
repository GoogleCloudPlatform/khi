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

package k8saudit_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	commonk8saudit "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// GCPK8sAuditLogExtractorTask provides K8sAuditLogExtractor for GCP audit logs.
var GCPK8sAuditLogExtractorTask = coretask.NewTask(
	k8saudit.GCPK8sAuditLogExtractorTaskID,
	[]coretask.Dependency{},
	func(ctx context.Context) (commonk8saudit.K8sAuditLogExtractor, error) {
		return k8saudit.ExtractGCPK8sAuditLog, nil
	},
	coretask.NewTaskResultRetentionLabel(true),
)

// GCPK8sAuditLogErrorExtractorTask provides K8sAuditLogErrorExtractor for GCP audit logs.
var GCPK8sAuditLogErrorExtractorTask = coretask.NewTask(
	k8saudit.GCPK8sAuditLogErrorExtractorTaskID,
	[]coretask.Dependency{},
	func(ctx context.Context) (commonk8saudit.K8sAuditLogErrorExtractor, error) {
		return k8saudit.ExtractGCPK8sAuditLogError, nil
	},
	coretask.NewTaskResultRetentionLabel(true),
)

var GCPK8sAuditLogParserTailTask = coretask.NewTailTask(
	k8saudit.GCPK8sAuditLogParserTailTaskID,
	[]coretask.Dependency{
		commonk8saudit.K8sAuditLogExtractorRef,
		commonk8saudit.K8sAuditLogErrorExtractorRef,
		commonk8saudit.NonSuccessLogLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.NamespaceRequestLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.ResourceRevisionLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.ConditionLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.ResourceOwnerReferenceTimelineMapperTaskID.Ref(),
		commonk8saudit.PodPhaseLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.EndpointResourceLogToTimelineMapperTaskID.Ref(),
		commonk8saudit.ContainerLogToTimelineMapperTaskID.Ref(),

		commonk8saudit.NodeNameDiscoveryTaskID.Ref(),
		commonk8saudit.ResourceUIDDiscoveryTaskID.Ref(),
		commonk8saudit.ContainerIDDiscoveryTaskID.Ref(),
		commonk8saudit.IPLeaseHistoryDiscoveryTaskID.Ref(),
		k8scommon.NEGNamesDiscoveryTaskID.Ref(),
		k8saudit.NEGToBackendServiceDiscoveryTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", `Gather Kubernetes audit logs to visualize resource modifications and API call histories on associated timelines.`, 1001, true),
)
