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

package onpremapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/onpremapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = gcpcommon.NewGCPOperationLogIngesterTask(
	onpremapiaudit.LogIngesterTaskID,
	onpremapiaudit.ListLogEntriesTaskID.Ref(),
	onpremapiaudit.LogTypeOnPremAPI,
)

// LogGrouperTask is a task that groups MulticloudAPI audit logs by their resource path.
// This grouping allows for parallel processing of logs related to the same resource.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	onpremapiaudit.LogGrouperTaskID,
	onpremapiaudit.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := onpremapiaudit.ExtractOnPremAPIAuditResource(l.NodeReader)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s/%s/%s", resourceFieldSet.Project, resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
	},
)

// OnPremAPIAuditTimelineMapper maps On-Prem API audit logs to timeline elements.
type OnPremAPIAuditTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*gcpcommon.GCPOperationTracker]
}

// LogIngesterTask returns the task reference providing the ingested logs.
func (m *OnPremAPIAuditTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return onpremapiaudit.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *OnPremAPIAuditTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *OnPremAPIAuditTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return onpremapiaudit.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps log entries to timeline elements.
func (m *OnPremAPIAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *gcpcommon.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *gcpcommon.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = gcpcommon.NewGCPOperationTracker()
	}
	auditFieldSet, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}
	resourceFieldSet, err := onpremapiaudit.ExtractOnPremAPIAuditResource(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	projectPath := gcpcommon.MustGCPProjectTimeline(ctx, resourceFieldSet.Project)
	clusterPath := onpremapiaudit.MustOnPremClusterTimeline(ctx, projectPath, resourceFieldSet.ClusterName)

	var targetPath *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetPath = clusterPath
	} else {
		targetPath = onpremapiaudit.MustOnPremNodePoolTimeline(ctx, clusterPath, resourceFieldSet.NodepoolName)
	}

	clusterTypeToFragmentInMethodNameMapping := map[onpremapiaudit.OnPremClusterType]string{
		onpremapiaudit.ClusterTypeBaremetalAdmin:      "BaremetalAdmin",
		onpremapiaudit.ClusterTypeBaremetalStandalone: "BaremetalStandalone",
		onpremapiaudit.ClusterTypeBaremetalUser:       "Baremetal",
		onpremapiaudit.ClusterTypeVMWareAdmin:         "VmwareAdmin",
		onpremapiaudit.ClusterTypeVMWareUser:          "Vmware",
	}

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]
	normalizedShortMethodName := strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "")

	operationPath := gcpcommon.MustGCPOperationTimeline(ctx, targetPath, shortMethodName, auditFieldSet.OperationID)
	gcpcommon.ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetPath, operationPath, &auditFieldSet, l.Timestamp, normalizedShortMethodName, resourceFieldSet.IsCluster())

	return cs, tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*gcpcommon.GCPOperationTracker] = (*OnPremAPIAuditTimelineMapper)(nil)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	onpremapiaudit.LogToTimelineMapperTaskID,
	&OnPremAPIAuditTimelineMapper{},
	inspectioncore.FeatureTaskLabel(
		"On-Premises API Logs",
		"Gather Anthos On-Premises audit logs to visualize cluster lifecycle events (creation, deletion, enrollment, unenrollment, and upgrades) on timelines.",
		9500,
		true,
	),
)
