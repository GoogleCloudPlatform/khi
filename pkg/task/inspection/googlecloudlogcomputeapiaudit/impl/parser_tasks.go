// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package googlecloudlogcomputeapiaudit_impl defines the implementation of the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_impl

import (
	"context"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask parses GCPOperationAuditLogFieldSet from raw Compute API logs.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID, googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
})

// LogIngesterTask is a V2 task that ingests log metadata (timestamp, severity, summary, log type) into KHI v6 format.
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudlogcomputeapiaudit_contract.LogIngesterTaskID,
	googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	googlecloudlogcomputeapiaudit_contract.LogTypeComputeApi,
)

// LogGrouperTask groups GCE API audit logs by node resource name for parallel mapper processing.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID, googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return getInstanceNameFromResourceName(audit.ResourceName)
	})

// LogToTimelineMapperTask maps GCE API audit logs to timeline events and revisions in parallel.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[*googlecloudcommon_contract.GCPOperationTracker](googlecloudlogcomputeapiaudit_contract.LogToTimelineMapperTaskID, &gcpComputeAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabelV2("Compute API Logs",
		"Gather Compute API audit logs to visualize the provisioning of infrastructure resources (e.g., GCE VM creation/deletion, Persistent Disk mounting) on associated timelines.",
		6000,
		true,
	),
)

type gcpComputeAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// LogIngesterTask returns a reference to the log ingester task.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the log grouper task.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup translates a single GCE API audit log into timeline event/revision changesets.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationTracker()
	}
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, tracker, err
	}
	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, tracker, err
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref())
	nodeTimelinePath := googlecloudlogcomputeapiaudit_contract.MustNodeTimelinePath(ctx, clusterIdentity.ClusterName, getInstanceNameFromResourceName(audit.ResourceName))

	var targetPath *khifilev6.TimelinePath
	if audit.ImmediateOperation() {
		targetPath = nodeTimelinePath
	} else {
		methodNameSplitted := strings.Split(audit.MethodName, ".")
		shortMethodName := "unknown"
		if len(methodNameSplitted) > 0 {
			shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
		}
		targetPath = googlecloudcommon_contract.MustGCPOperationTimeline(ctx, nodeTimelinePath, shortMethodName, audit.OperationID)
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	tracker.ProcessOperationLog(ctx, cs, targetPath, audit, commonLogFieldSet.Timestamp)

	return cs, tracker, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogToTimelineMapperV2[*googlecloudcommon_contract.GCPOperationTracker] = (*gcpComputeAuditLogLogToTimelineMapperSetting)(nil)

func getInstanceNameFromResourceName(resourceName string) string {
	resourceNameSplitted := strings.Split(resourceName, "/")
	if len(resourceNameSplitted) < 1 {
		return ""
	}
	return resourceNameSplitted[len(resourceNameSplitted)-1]
}
