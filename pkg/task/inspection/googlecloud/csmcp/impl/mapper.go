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

package csmcp_impl

import (
	"context"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
)

var (
	pathContainerName = structured.CompileFieldPath("resource.labels.container_name")
	pathPodName       = structured.CompileFieldPath("resource.labels.pod_name")
)

// istiodLogFilterTask filters container logs to Istiod discovery logs.
var istiodLogFilterTask = inspectiontaskbase.DefineLogFilterTask(
	csmcp.IstiodLogFilterTaskID,
	k8scontainer.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			return l.NodeReader.ReadStringOrDefault(pathContainerName, "") == "discovery" &&
				strings.Contains(l.NodeReader.ReadStringOrDefault(pathPodName, ""), "istiod")
		}
	},
)

// logGrouperTask groups Istiod discovery logs by a constant key "csmcp" to process all logs in chronological order.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	csmcp.LogGrouperTaskID,
	csmcp.IstiodLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			return "csmcp"
		}
	},
)

// csmcpTimelineMapper maps Istiod discovery logs to the CSM control plane timelines of the pods they mention.
type csmcpTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*commoncsmcp.TimelineState]
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup maps a log entry to its corresponding timeline paths.
func (m *csmcpTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *commoncsmcp.TimelineState) (*khifilev6.TimelineChangeSet, *commoncsmcp.TimelineState, error) {
	return mapCSMCPLog(ctx, l, prevGroupData, m.clusterIdentity.Get(ctx))
}

var _ inspectiontaskbase.TimelineMapper[*commoncsmcp.TimelineState] = (*csmcpTimelineMapper)(nil)

// mapCSMCPLog maps an Istiod discovery log to the pod and connection timelines of the pods it mentions.
// It uses the cluster name in the log when clusterIdentity has no cluster name.
func mapCSMCPLog(ctx context.Context, l *log.Log, prevGroupData *commoncsmcp.TimelineState, clusterIdentity k8scommon.GoogleCloudClusterIdentity) (*khifilev6.TimelineChangeSet, *commoncsmcp.TimelineState, error) {
	fs, err := csmcp.Extract(l.NodeReader)
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

// logToTimelineMapperTask is the task that maps in-cluster Control Plane Logs to timelines.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	csmcp.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8scontainer.LogIngesterTaskID.Ref(),
		GroupedLogs: csmcp.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*commoncsmcp.TimelineState] {
		return &csmcpTimelineMapper{
			clusterIdentity: coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref()),
		}
	},
)
