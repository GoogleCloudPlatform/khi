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

package k8snode_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
)

// OtherLogFilterTask filters only the components logs that do not match kubelet or containerd.
var OtherLogFilterTask = newParserTypeFilterTask(k8snode.OtherLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Other)

// OtherLogGroupTask groups other logs by node and component.
var OtherLogGroupTask = newNodeAndComponentNameGrouperTask(k8snode.OtherLogGroupTaskID, k8snode.OtherLogFilterTaskID.Ref())

type otherNodeLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
	StartingMessagesByComponent    map[string]string
	TerminatingMessagesByComponent map[string]string
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8snode.ClusterIdentityTaskID.Ref(),
		k8snode.PodSandboxIDDiscoveryTaskID.Ref(),
		k8saudit.ContainerIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8snode.OtherLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8snode.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *otherNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, k8snode.ClusterIdentityTaskID.Ref())
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := MustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	var startingMessage string
	var terminatingMessage string
	if msg, found := o.StartingMessagesByComponent[componentFieldSet.Component]; found {
		startingMessage = msg
	}
	if msg, found := o.TerminatingMessagesByComponent[componentFieldSet.Component]; found {
		terminatingMessage = msg
	}
	checkStartingAndTerminationLog(ctx, cs, l, startingMessage, terminatingMessage, componentTimelinePath)

	cs.AddEvent(componentTimelinePath)

	if componentFieldSet.Message != nil {
		podSandboxIDFinder := coretask.GetTaskResult(ctx, k8snode.PodSandboxIDDiscoveryTaskID.Ref())
		containerIDPatternFinder := coretask.GetTaskResult(ctx, k8saudit.ContainerIDPatternFinderTaskID.Ref())
		pods, containerRefs := findPodAndContainerReferences(componentFieldSet.Message.Raw(), podSandboxIDFinder, containerIDPatternFinder)
		for _, pod := range pods {
			cs.AddEvent(MustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName))
		}
		for _, containerRef := range containerRefs {
			podTimelinePath := MustK8sPodTimeline(ctx, clusterName, containerRef.Pod.PodNamespace, containerRef.Pod.PodName)
			cs.AddEvent(k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, containerRef.Container.ContainerName))
		}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*otherNodeLogLogToTimelineMapperSetting)(nil)

// OtherLogLogToTimelineMapperTask registers the mapper for other node component logs.
var OtherLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(k8snode.OtherLogLogToTimelineMapperTaskID, &otherNodeLogLogToTimelineMapperSetting{
	StartingMessagesByComponent: map[string]string{
		"dockerd":             "Starting up",
		"configure.sh":        "Start to install kubernetes files",
		"configure-helper.sh": "Start to configure instance for kubernetes",
	},
	TerminatingMessagesByComponent: map[string]string{
		"dockerd":             "Daemon shutdown complete",
		"configure.sh":        "Done for installing kubernetes files",
		"configure-helper.sh": "Done for the configuration for kubernetes",
	},
})
