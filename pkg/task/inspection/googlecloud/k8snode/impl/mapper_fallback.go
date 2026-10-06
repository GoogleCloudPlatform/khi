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

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
)

// otherLogFilterTask filters only the components logs that do not match kubelet or containerd.
var otherLogFilterTask = defineParserTypeFilterTask(k8snode.OtherLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Other)

// otherLogGroupTask groups other logs by node and component.
var otherLogGroupTask = defineNodeAndComponentNameGrouperTask(k8snode.OtherLogGroupTaskID, k8snode.OtherLogFilterTaskID.Ref())

// otherLogTimelineMapper maps the logs of node components other than containerd and kubelet to the timelines of their component and of the pods and containers they mention.
type otherLogTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	// startingMessagesByComponent maps a component name to the log message that marks the start of the component.
	startingMessagesByComponent map[string]string
	// terminatingMessagesByComponent maps a component name to the log message that marks the termination of the component.
	terminatingMessagesByComponent map[string]string
	clusterIdentity                coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	podSandboxIDFinder             coretask.Input[patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo]]
	containerIDPatternFinder       coretask.Input[patternfinder.PatternFinder[*k8saudit.ContainerIdentity]]
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (o *otherLogTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := o.mapLog(ctx, l, o.clusterIdentity.Get(ctx), o.podSandboxIDFinder.Get(ctx), o.containerIDPatternFinder.Get(ctx))
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*otherLogTimelineMapper)(nil)

// mapLog adds events for a node component log to the timeline of its component and to the timelines of the pods and containers it mentions.
// It also adds a revision to the component timeline when the log message is the starting or terminating message of the component.
func (o *otherLogTimelineMapper) mapLog(ctx context.Context, l *log.Log, clusterIdentity k8scommon.GoogleCloudClusterIdentity, podSandboxIDFinder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], containerIDPatternFinder patternfinder.PatternFinder[*k8saudit.ContainerIdentity]) (*khifilev6.TimelineChangeSet, error) {
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := mustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	var startingMessage string
	var terminatingMessage string
	if msg, found := o.startingMessagesByComponent[componentFieldSet.Component]; found {
		startingMessage = msg
	}
	if msg, found := o.terminatingMessagesByComponent[componentFieldSet.Component]; found {
		terminatingMessage = msg
	}
	checkStartingAndTerminationLog(ctx, cs, l, startingMessage, terminatingMessage, componentTimelinePath)

	cs.AddEvent(componentTimelinePath)

	if componentFieldSet.Message != nil {
		pods, containerRefs := findPodAndContainerReferences(componentFieldSet.Message.Raw(), podSandboxIDFinder, containerIDPatternFinder)
		for _, pod := range pods {
			cs.AddEvent(mustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName))
		}
		for _, containerRef := range containerRefs {
			podTimelinePath := mustK8sPodTimeline(ctx, clusterName, containerRef.Pod.PodNamespace, containerRef.Pod.PodName)
			cs.AddEvent(k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, containerRef.Container.ContainerName))
		}
	}

	return cs, nil
}

// otherLogLogToTimelineMapperTask maps the logs of the other node components to timelines.
var otherLogLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8snode.OtherLogLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8snode.LogIngesterTaskID.Ref(),
		GroupedLogs: k8snode.OtherLogGroupTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &otherLogTimelineMapper{
			startingMessagesByComponent: map[string]string{
				"dockerd":             "Starting up",
				"configure.sh":        "Start to install kubernetes files",
				"configure-helper.sh": "Start to configure instance for kubernetes",
			},
			terminatingMessagesByComponent: map[string]string{
				"dockerd":             "Daemon shutdown complete",
				"configure.sh":        "Done for installing kubernetes files",
				"configure-helper.sh": "Done for the configuration for kubernetes",
			},
			clusterIdentity:          coretask.Use(b, k8snode.ClusterIdentityTaskID.Ref()),
			podSandboxIDFinder:       coretask.Use(b, k8snode.PodSandboxIDDiscoveryTaskID.Ref()),
			containerIDPatternFinder: coretask.Use(b, k8saudit.ContainerIDPatternFinderTaskID.Ref()),
		}
	},
)
