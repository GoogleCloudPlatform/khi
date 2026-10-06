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

package k8scontrolplane_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
)

// schedulerLogFilterTask filters kube-scheduler logs.
var schedulerLogFilterTask = inspectiontaskbase.DefineLogFilterTask(
	k8scontrolplane.SchedulerLogFilterTaskID,
	k8scontrolplane.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			parserType, err := k8scontrolplane.ExtractK8sControlplaneComponentParserType(l.NodeReader)
			if err != nil {
				return false
			}
			return parserType == k8scontrolplane.ComponentParserTypeScheduler
		}
	},
)

// schedulerLogGrouperTask puts all kube-scheduler logs into one group.
var schedulerLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	k8scontrolplane.SchedulerLogGrouperTaskID,
	k8scontrolplane.SchedulerLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			return "" // No grouping needed
		}
	},
)

// schedulerTimelineMapper maps scheduler logs to timeline paths.
type schedulerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (m *schedulerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	componentFieldSet, err := k8scontrolplane.ExtractK8sControlplaneComponent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}
	schedulerMessageFieldSet, err := k8scontrolplane.ExtractK8sSchedulerComponent(l.NodeReader, nil)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, componentFieldSet.ProjectID)
	gkeTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, componentFieldSet.ClusterName)
	compTimeline := k8scontrolplane.MustControlPlaneComponentTimeline(ctx, gkeTimeline, componentFieldSet.ComponentName)
	cs.AddEvent(compTimeline)

	if schedulerMessageFieldSet.HasPodField() {
		clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, componentFieldSet.ClusterName)
		apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
		kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
		namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, schedulerMessageFieldSet.PodNamespace)
		podTimeline := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, schedulerMessageFieldSet.PodName)
		cs.AddEvent(podTimeline)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*schedulerTimelineMapper)(nil)

// schedulerLogToTimelineMapperTask maps kube-scheduler logs to the timelines of the scheduler and the Pods they mention.
var schedulerLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8scontrolplane.SchedulerLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8scontrolplane.LogIngesterTaskID.Ref(),
		GroupedLogs: k8scontrolplane.SchedulerLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &schedulerTimelineMapper{}
	},
)
