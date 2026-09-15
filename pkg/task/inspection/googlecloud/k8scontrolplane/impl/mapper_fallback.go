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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
)

var OtherLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	k8scontrolplane.OtherLogFilterTaskID,
	k8scontrolplane.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		parserType, err := k8scontrolplane.ExtractK8sControlplaneComponentParserType(l.NodeReader)
		if err != nil {
			return false
		}
		return parserType == k8scontrolplane.ComponentParserTypeOther
	},
)

var OtherGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	k8scontrolplane.OtherLogGrouperTaskID,
	k8scontrolplane.OtherLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		componentFieldSet, err := k8scontrolplane.ExtractK8sControlplaneComponent(l.NodeReader)
		if err != nil {
			return ""
		}
		return componentFieldSet.ComponentName
	},
)

// OtherTimelineMapper maps other control plane logs to timeline paths.
type OtherTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8scontrolplane.OtherLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8scontrolplane.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	componentFieldSet, err := k8scontrolplane.ExtractK8sControlplaneComponent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, componentFieldSet.ProjectID)
	gkeTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, componentFieldSet.ClusterName)
	compTimeline := k8scontrolplane.MustControlPlaneComponentTimeline(ctx, gkeTimeline, componentFieldSet.ComponentName)
	cs.AddEvent(compTimeline)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*OtherTimelineMapper)(nil)

var OtherLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](k8scontrolplane.OtherLogToTimelineMapperTaskID, &OtherTimelineMapper{})
