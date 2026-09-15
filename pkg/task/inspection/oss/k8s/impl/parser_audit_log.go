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

package ossk8s_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	ossk8s "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/oss/k8s"
)

// OSSK8sAuditLogExtractorTask provides K8sAuditLogExtractor for OSS audit logs.
var OSSK8sAuditLogExtractorTask = coretask.NewTask(
	ossk8s.OSSK8sAuditLogExtractorTaskID,
	[]coretask.Dependency{},
	func(ctx context.Context) (k8saudit.K8sAuditLogExtractor, error) {
		return ossk8s.ExtractOSSK8sAuditLog, nil
	},
	coretask.NewTaskResultRetentionLabel(true),
)

// OSSK8sAuditLogErrorExtractorTask provides K8sAuditLogErrorExtractor for OSS audit logs.
var OSSK8sAuditLogErrorExtractorTask = coretask.NewTask(
	ossk8s.OSSK8sAuditLogErrorExtractorTaskID,
	[]coretask.Dependency{},
	func(ctx context.Context) (k8saudit.K8sAuditLogErrorExtractor, error) {
		return ossk8s.ExtractOSSK8sAuditLogError, nil
	},
	coretask.NewTaskResultRetentionLabel(true),
)

var OSSK8sAuditLogParserTailTask = coretask.NewTailTask(
	ossk8s.OSSK8sAuditLogParserTailTaskID,
	[]coretask.Dependency{
		k8saudit.K8sAuditLogExtractorRef,
		k8saudit.K8sAuditLogErrorExtractorRef,
		k8saudit.NonSuccessLogLogToTimelineMapperTaskID.Ref(),
		k8saudit.NamespaceRequestLogToTimelineMapperTaskID.Ref(),
		k8saudit.ResourceRevisionLogToTimelineMapperTaskID.Ref(),
		k8saudit.ConditionLogToTimelineMapperTaskID.Ref(),
		k8saudit.ResourceOwnerReferenceTimelineMapperTaskID.Ref(),
		k8saudit.PodPhaseLogToTimelineMapperTaskID.Ref(),
		k8saudit.EndpointResourceLogToTimelineMapperTaskID.Ref(),
		k8saudit.ContainerLogToTimelineMapperTaskID.Ref(),

		k8saudit.NodeNameDiscoveryTaskID.Ref(),
		k8saudit.ResourceUIDDiscoveryTaskID.Ref(),
		k8saudit.ContainerIDDiscoveryTaskID.Ref(),
		k8saudit.IPLeaseHistoryDiscoveryTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", `Gather Kubernetes audit logs to visualize resource modifications and API call histories on associated timelines.`, 1001, true),
)
