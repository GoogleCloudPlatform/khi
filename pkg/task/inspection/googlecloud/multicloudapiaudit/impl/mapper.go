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

package multicloudapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/multicloudapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// logIngesterTask serializes MulticloudAPI audit logs for storage in the history builder.
var logIngesterTask = gcpcommon.DefineGCPOperationLogIngesterTask(
	multicloudapiaudit.LogIngesterTaskID,
	multicloudapiaudit.ListLogEntriesTaskID.Ref(),
	multicloudapiaudit.LogTypeMulticloudAPI,
)

// logGrouperTask groups MulticloudAPI audit logs by resource identifier.
// This grouping allows for parallel processing of logs related to the same resource.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	multicloudapiaudit.LogGrouperTaskID,
	multicloudapiaudit.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			resourceFieldSet, err := multicloudapiaudit.ExtractMulticloudAPIAuditResource(l.NodeReader)
			if err != nil {
				return ""
			}
			if resourceFieldSet.IsCluster() {
				return fmt.Sprintf("cluster/%s/%s", resourceFieldSet.ClusterType, resourceFieldSet.ClusterName)
			}
			return fmt.Sprintf("nodepool/%s/%s/%s", resourceFieldSet.ClusterType, resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
		}
	},
)

// multiCloudAuditTimelineMapper maps grouped logs to resource timelines and operations in KHI V6 format.
type multiCloudAuditTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*gcpcommon.GCPOperationTracker]
}

var _ inspectiontaskbase.TimelineMapper[*gcpcommon.GCPOperationTracker] = (*multiCloudAuditTimelineMapper)(nil)

// ProcessLogByGroup maps grouped logs to resource timelines and operations in KHI V6 format.
func (m *multiCloudAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *gcpcommon.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *gcpcommon.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = gcpcommon.NewGCPOperationTracker()
	}
	auditFieldSet, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}
	resourceFieldSet, err := multicloudapiaudit.ExtractMulticloudAPIAuditResource(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	projectPath := gcpcommon.MustGCPProjectTimeline(ctx, auditFieldSet.ProjectID)
	clusterPath := multicloudapiaudit.MustMultiCloudClusterTimeline(ctx, projectPath, resourceFieldSet.ClusterName)

	var targetPath *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetPath = clusterPath
	} else {
		targetPath = multicloudapiaudit.MustMultiCloudNodepoolTimeline(ctx, clusterPath, resourceFieldSet.NodepoolName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	clusterTypeToFragmentInMethodNameMapping := map[multicloudapiaudit.MultiCloudClusterType]string{
		multicloudapiaudit.ClusterTypeAWS:   "Aws",
		multicloudapiaudit.ClusterTypeAzure: "Azure",
	}

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]
	shortMethodName = strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "") // Remove type specific part.

	opPath := multicloudapiaudit.MustOperationTimeline(ctx, targetPath, shortMethodName, auditFieldSet.OperationID)
	gcpcommon.ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetPath, opPath, &auditFieldSet, l.Timestamp, shortMethodName, resourceFieldSet.IsCluster())

	return cs, tracker, nil
}

// logToTimelineMapperTask adds revisions/events regarding logs.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	multicloudapiaudit.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: multicloudapiaudit.LogIngesterTaskID.Ref(),
		GroupedLogs: multicloudapiaudit.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*gcpcommon.GCPOperationTracker] {
		return &multiCloudAuditTimelineMapper{}
	},
	inspectioncore.FeatureTaskLabel(`Multi-Cloud API Logs`,
		`Gather Anthos Multi-Cloud audit logs to visualize cluster lifecycle events (creation, deletion, and upgrades) on timelines.`,
		5000,
		true,
	),
)
