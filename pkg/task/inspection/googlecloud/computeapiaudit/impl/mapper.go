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

// Package computeapiaudit_impl defines the implementation of compute API audit inspection tasks.
package computeapiaudit_impl

import (
	"context"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/computeapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// logIngesterTask is a task that ingests log metadata (timestamp, severity, summary, log type) into KHI v6 format.
var logIngesterTask = gcpcommon.DefineGCPOperationLogIngesterTask(
	computeapiaudit.LogIngesterTaskID,
	computeapiaudit.ListLogEntriesTaskID.Ref(),
	computeapiaudit.LogTypeComputeApi,
)

// logGrouperTask groups GCE API audit logs by node resource name for parallel mapper processing.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	computeapiaudit.LogGrouperTaskID,
	computeapiaudit.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
			if err != nil {
				return "unknown"
			}
			return getInstanceNameFromResourceName(audit.ResourceName)
		}
	},
)

// computeAuditTimelineMapper maps grouped GCE API audit logs to node and operation timelines.
type computeAuditTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*gcpcommon.GCPOperationTracker]
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup translates a single GCE API audit log into timeline event/revision changesets.
func (m *computeAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *gcpcommon.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *gcpcommon.GCPOperationTracker, error) {
	return mapComputeAuditLog(ctx, l, tracker, m.clusterIdentity.Get(ctx).ClusterName)
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.TimelineMapper[*gcpcommon.GCPOperationTracker] = (*computeAuditTimelineMapper)(nil)

// mapComputeAuditLog translates a single GCE API audit log into timeline event/revision changesets for the given cluster.
func mapComputeAuditLog(ctx context.Context, l *log.Log, tracker *gcpcommon.GCPOperationTracker, clusterName string) (*khifilev6.TimelineChangeSet, *gcpcommon.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = gcpcommon.NewGCPOperationTracker()
	}
	audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	nodeTimelinePath := computeapiaudit.MustNodeTimelinePath(ctx, clusterName, getInstanceNameFromResourceName(audit.ResourceName))

	var targetPath *khifilev6.TimelinePath
	if audit.ImmediateOperation() {
		targetPath = nodeTimelinePath
	} else {
		methodNameSplitted := strings.Split(audit.MethodName, ".")
		shortMethodName := "unknown"
		if len(methodNameSplitted) > 0 {
			shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
		}
		targetPath = gcpcommon.MustGCPOperationTimeline(ctx, nodeTimelinePath, shortMethodName, audit.OperationID)
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	tracker.ProcessOperationLog(ctx, cs, targetPath, &audit, l.Timestamp)

	return cs, tracker, nil
}

// logToTimelineMapperTask maps GCE API audit logs to timeline events and revisions in parallel.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	computeapiaudit.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: computeapiaudit.LogIngesterTaskID.Ref(),
		GroupedLogs: computeapiaudit.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*gcpcommon.GCPOperationTracker] {
		return &computeAuditTimelineMapper{
			clusterIdentity: coretask.Use(b, computeapiaudit.ClusterIdentityTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel("Compute API Logs",
		"Gather Compute API audit logs to visualize the provisioning of infrastructure resources (e.g., GCE VM creation/deletion, Persistent Disk mounting) on associated timelines.",
		6000,
		true,
	),
)

func getInstanceNameFromResourceName(resourceName string) string {
	resourceNameSplitted := strings.Split(resourceName, "/")
	if len(resourceNameSplitted) < 1 {
		return ""
	}
	return resourceNameSplitted[len(resourceNameSplitted)-1]
}
