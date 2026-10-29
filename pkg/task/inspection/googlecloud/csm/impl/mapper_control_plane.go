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

package csm_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// ingestCSMControlPlaneLog ingests a single CSM control plane log entry into a LogChangeSet.
func ingestCSMControlPlaneLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(commoncsmcp.LogTypeCSMCP)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	}

	fs, err := csm.ExtractCSMControlPlaneLog(l.NodeReader)
	if err != nil {
		return nil, err
	}
	cs.SetSummary(fs.Message)
	if fs.Timestamp != nil {
		cs.SetTimestamp(*fs.Timestamp)
	}

	return cs, nil
}

// csmControlPlaneLogIngesterTask ingests CSM control plane logs.
var csmControlPlaneLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	csm.CSMControlPlaneLogIngesterTaskID,
	csm.ListCSMControlPlaneLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return ingestCSMControlPlaneLog
	},
)

// groupCSMControlPlaneLog groups CSM control plane logs by project, location, and revision.
func groupCSMControlPlaneLog(ctx context.Context, l *log.Log) string {
	fs, err := csm.ExtractCSMControlPlaneLog(l.NodeReader)
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%s/%s/%s", fs.ProjectID, fs.Location, fs.Revision)
}

// csmControlPlaneLogGrouperTask groups CSM control plane logs.
var csmControlPlaneLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	csm.CSMControlPlaneLogGrouperTaskID,
	csm.ListCSMControlPlaneLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return groupCSMControlPlaneLog
	},
)

// csmControlPlaneTimelineMapper maps CSM control plane logs to resource timelines.
type csmControlPlaneTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*commoncsmcp.TimelineState]
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup maps each control plane log inside a group to its control plane, pod, and connection timelines.
func (m *csmControlPlaneTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *commoncsmcp.TimelineState) (*khifilev6.TimelineChangeSet, *commoncsmcp.TimelineState, error) {
	return mapCSMControlPlaneLog(ctx, l, prevGroupData, m.clusterIdentity.Get(ctx))
}

var _ inspectiontaskbase.TimelineMapper[*commoncsmcp.TimelineState] = (*csmControlPlaneTimelineMapper)(nil)

// mapCSMControlPlaneLog maps a CSM control plane log entry to its control plane, pod, and connection timeline paths.
func mapCSMControlPlaneLog(
	ctx context.Context,
	l *log.Log,
	prevGroupData *commoncsmcp.TimelineState,
	clusterIdentity k8scommon.GoogleCloudClusterIdentity,
) (*khifilev6.TimelineChangeSet, *commoncsmcp.TimelineState, error) {
	fs, err := csm.ExtractCSMControlPlaneLog(l.NodeReader)
	if err != nil {
		return nil, prevGroupData, err
	}
	cs := khifilev6.NewTimelineChangeSet(l)

	timelinePath := csm.MustCSMControlPlaneTimeline(ctx)
	cs.AddEvent(timelinePath)

	changedTime := l.Timestamp
	if fs.Timestamp != nil {
		changedTime = *fs.Timestamp
	}

	if commoncsmcp.IsXDSLog(fs.Message) && clusterIdentity.ClusterName != "" && len(fs.Pods) > 0 {
		nextGroupData := commoncsmcp.MapPodAndConnectionTimelines(
			ctx,
			cs,
			clusterIdentity.ClusterName,
			fs.Message,
			changedTime,
			fs.Pods,
			prevGroupData,
		)
		return cs, nextGroupData, nil
	}

	return cs, prevGroupData, nil
}

// csmControlPlaneLogToTimelineMapperTask maps CSM control plane logs to timelines.
var csmControlPlaneLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	csm.CSMControlPlaneLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: csm.CSMControlPlaneLogIngesterTaskID.Ref(),
		GroupedLogs: csm.CSMControlPlaneLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*commoncsmcp.TimelineState] {
		return &csmControlPlaneTimelineMapper{
			clusterIdentity: coretask.Use(b, csm.ClusterIdentityTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"CSM Control Plane Logs",
		"Gather Managed Cloud Service Mesh control plane (istiod) logs to visualize xDS configuration pushes and proxy connections on associated Pod timelines.",
		10200,
		false,
	),
)
