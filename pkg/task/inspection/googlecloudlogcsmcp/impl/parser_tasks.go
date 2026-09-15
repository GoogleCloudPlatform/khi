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

package googlecloudlogcsmcp_impl

import (
	"context"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	googlecloudlogcsmcp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsmcp/contract"
)

var (
	pathContainerName = structured.CompileFieldPath("resource.labels.container_name")
	pathPodName       = structured.CompileFieldPath("resource.labels.pod_name")
)

// IstiodLogFilterTask filters container logs to Istiod discovery logs.
var IstiodLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogcsmcp_contract.IstiodLogFilterTaskID,
	k8scontainer.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		return l.NodeReader.ReadStringOrDefault(pathContainerName, "") == "discovery" &&
			strings.Contains(l.NodeReader.ReadStringOrDefault(pathPodName, ""), "istiod")
	},
)

// LogGrouperTask groups Istiod discovery logs by a constant key "csmcp" to process all logs in chronological order.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogcsmcp_contract.LogGrouperTaskID,
	googlecloudlogcsmcp_contract.IstiodLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return "csmcp"
	},
)

type csmcpTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*commoncsmcp.TimelineState]
}

// LogIngesterTask returns the task reference for log ingestion.
func (m *csmcpTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8scontainer.LogIngesterTaskID.Ref()
}

// Dependencies returns dependencies needed for mapping.
func (m *csmcpTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns the task reference for grouped logs.
func (m *csmcpTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcsmcp_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps a log entry to its corresponding timeline paths.
func (m *csmcpTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *commoncsmcp.TimelineState) (*khifilev6.TimelineChangeSet, *commoncsmcp.TimelineState, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())

	fs, err := googlecloudlogcsmcp_contract.Extract(l.NodeReader)
	if err != nil {
		return nil, prevGroupData, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	clusterName := clusterIdentity.ClusterName
	if clusterName == "" {
		clusterName = fs.ClusterName
	}

	var changedTime time.Time
	if fs.Timestamp != nil {
		changedTime = *fs.Timestamp
	} else {
		changedTime = l.Timestamp
	}

	nextGroupData := commoncsmcp.MapPodAndConnectionTimelines(
		ctx,
		cs,
		clusterName,
		fs.Message,
		changedTime,
		fs.Pods,
		prevGroupData,
	)

	return cs, nextGroupData, nil
}

// LogToTimelineMapperTask is the task that maps in-cluster Control Plane Logs to timelines.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudlogcsmcp_contract.LogToTimelineMapperTaskID,
	&csmcpTimelineMapper{},
)

var _ inspectiontaskbase.LogToTimelineMapper[*commoncsmcp.TimelineState] = (*csmcpTimelineMapper)(nil)
