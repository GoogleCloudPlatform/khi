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
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
)

// kubeletLogFilterTask filters only kubelet component logs.
var kubeletLogFilterTask = defineParserTypeFilterTask(k8snode.KubeletLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Kubelet)

// kubeletLogGroupTask groups kubelet logs by node and component.
var kubeletLogGroupTask = defineNodeAndComponentNameGrouperTask(k8snode.KubeletLogGroupTaskID, k8snode.KubeletLogFilterTaskID.Ref())

// kubeletLogTimelineMapper maps kubelet logs to the timelines of their node component and of the pods, containers and resources they mention.
type kubeletLogTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity          coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	podSandboxIDFinder       coretask.Input[patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo]]
	containerIDPatternFinder coretask.Input[patternfinder.PatternFinder[*k8saudit.ContainerIdentity]]
	resourceUIDPatternFinder coretask.Input[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]]
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (k *kubeletLogTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapKubeletLog(ctx, l, k.clusterIdentity.Get(ctx), k.podSandboxIDFinder.Get(ctx), k.containerIDPatternFinder.Get(ctx), k.resourceUIDPatternFinder.Get(ctx))
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*kubeletLogTimelineMapper)(nil)

// mapKubeletLog adds events for a kubelet log to the timeline of its node component and to the timelines of the pods, containers and resources it mentions.
func mapKubeletLog(ctx context.Context, l *log.Log, clusterIdentity k8scommon.GoogleCloudClusterIdentity, podSandboxIDFinder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], containerIDPatternFinder patternfinder.PatternFinder[*k8saudit.ContainerIdentity], resourceUIDPatternFinder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.TimelineChangeSet, error) {
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := mustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	cs.AddEvent(componentTimelinePath)

	original := componentFieldSet.Message.Raw()

	foundPods := map[string]struct{}{}
	pods, containerRefs := findPodAndContainerReferences(original, podSandboxIDFinder, containerIDPatternFinder)
	for _, pod := range pods {
		cs.AddEvent(mustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName))
		foundPods[fmt.Sprintf("%s/%s", pod.PodNamespace, pod.PodName)] = struct{}{}
	}
	for _, containerRef := range containerRefs {
		podTimelinePath := mustK8sPodTimeline(ctx, clusterName, containerRef.Pod.PodNamespace, containerRef.Pod.PodName)
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
			podTimelinePath := mustK8sPodTimeline(ctx, clusterName, podNamespace, podName)
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
					podTimelinePath := mustK8sPodTimeline(ctx, clusterName, podNamespace, podName)
					cs.AddEvent(podTimelinePath)
				}
			}
		}
	}

	return cs, nil
}

// kubeletLogLogToTimelineMapperTask maps kubelet logs to timelines.
var kubeletLogLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8snode.KubeletLogLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8snode.LogIngesterTaskID.Ref(),
		GroupedLogs: k8snode.KubeletLogGroupTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &kubeletLogTimelineMapper{
			clusterIdentity:          coretask.Use(b, k8snode.ClusterIdentityTaskID.Ref()),
			podSandboxIDFinder:       coretask.Use(b, k8snode.PodSandboxIDDiscoveryTaskID.Ref()),
			containerIDPatternFinder: coretask.Use(b, k8saudit.ContainerIDPatternFinderTaskID.Ref()),
			resourceUIDPatternFinder: coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref()),
		}
	},
)
