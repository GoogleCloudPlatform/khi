// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses///     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8snode_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
)

// KubeletLogFilterTask filters only kubelet component logs.
var KubeletLogFilterTask = newParserTypeFilterTask(k8snode.KubeletLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Kubelet)

// KubeletLogGroupTask groups kubelet logs by node and component.
var KubeletLogGroupTask = newNodeAndComponentNameGrouperTask(k8snode.KubeletLogGroupTaskID, k8snode.KubeletLogFilterTaskID.Ref())

type kubeletNodeLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8snode.ClusterIdentityTaskID.Ref(),
		k8snode.PodSandboxIDDiscoveryTaskID.Ref(),
		k8saudit.ContainerIDPatternFinderTaskID.Ref(),
		k8saudit.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8snode.KubeletLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8snode.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, k8snode.ClusterIdentityTaskID.Ref())
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, struct{}{}, err
	}
	containerIDPatternFinder := coretask.GetTaskResult(ctx, k8saudit.ContainerIDPatternFinderTaskID.Ref())
	podSandboxIDFinder := coretask.GetTaskResult(ctx, k8snode.PodSandboxIDDiscoveryTaskID.Ref())
	resourceUIDPatternFinder := coretask.GetTaskResult(ctx, k8saudit.ResourceUIDPatternFinderTaskID.Ref())

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := MustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	cs.AddEvent(componentTimelinePath)

	original := componentFieldSet.Message.Raw()

	foundPods := map[string]struct{}{}
	pods, containerRefs := findPodAndContainerReferences(original, podSandboxIDFinder, containerIDPatternFinder)
	for _, pod := range pods {
		cs.AddEvent(MustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName))
		foundPods[fmt.Sprintf("%s/%s", pod.PodNamespace, pod.PodName)] = struct{}{}
	}
	for _, containerRef := range containerRefs {
		podTimelinePath := MustK8sPodTimeline(ctx, clusterName, containerRef.Pod.PodNamespace, containerRef.Pod.PodName)
		cs.AddEvent(k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, containerRef.Container.ContainerName))
	}

	resourceFindResults := patternfinder.FindAllWithStarterRunes(original, resourceUIDPatternFinder, false, '"')
	for _, result := range resourceFindResults {
		res := result.Value
		if res.APIVersion == "core/v1" && res.Kind == "pod" {
			if _, ok := foundPods[fmt.Sprintf("%s/%s", res.Namespace, res.Name)]; ok {
				continue
			}
		}
		resTimelinePath := k8saudit.MustResourceTimeline(ctx, clusterName, res)
		cs.AddEvent(resTimelinePath)
	}

	// Kubelet specific resource bindings
	// When this log can't be associated with resource by container id or pod sandbox id, try to get it from klog fields.
	podNameWithNamespace, err := componentFieldSet.Message.StringField("pod")
	if err == nil && podNameWithNamespace != "" {
		podNamespace, podName, err := slashSplittedPodNameToNamespaceAndName(podNameWithNamespace)
		if err == nil {
			podTimelinePath := MustK8sPodTimeline(ctx, clusterName, podNamespace, podName)
			containerName, err := componentFieldSet.Message.StringField("containerName")
			if err == nil && containerName != "" {
				containerTimelinePath := k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, containerName)
				cs.AddEvent(containerTimelinePath)
			} else {
				cs.AddEvent(podTimelinePath)
			}
		}
	} else {
		podNames, err := componentFieldSet.Message.StringField("pods")
		if err == nil && podNames != "" {
			podNames = strings.Trim(podNames, "[]")
			podNamesSplitted := strings.Split(podNames, ",")
			for _, podNamespaceAndNameWithSlash := range podNamesSplitted {
				podNamespaceAndNameWithSlash = strings.Trim(podNamespaceAndNameWithSlash, `"`)
				podNamespace, podName, err := slashSplittedPodNameToNamespaceAndName(podNamespaceAndNameWithSlash)
				if err == nil {
					podTimelinePath := MustK8sPodTimeline(ctx, clusterName, podNamespace, podName)
					cs.AddEvent(podTimelinePath)
				}
			}
		}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*kubeletNodeLogLogToTimelineMapperSetting)(nil)

// KubeletLogLogToTimelineMapperTask registers the mapper for kubelet component logs.
var KubeletLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	k8snode.KubeletLogLogToTimelineMapperTaskID,
	&kubeletNodeLogLogToTimelineMapperSetting{},
)
