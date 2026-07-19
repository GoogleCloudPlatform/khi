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

package googlecloudlogk8snode_impl

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
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

// KubeletLogFilterTask filters only kubelet component logs.
var KubeletLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.KubeletLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Kubelet)

// KubeletLogGroupTask groups kubelet logs by node and component.
var KubeletLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.KubeletLogGroupTaskID, googlecloudlogk8snode_contract.KubeletLogFilterTaskID.Ref())

type kubeletNodeLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref(),
		googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref(),
		commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.KubeletLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (k *kubeletNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref())
	clusterName := clusterIdentity.NameFor(googlecloudk8scommon_contract.ClusterNameUsageK8sCluster)
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	containerIDPatternFinder := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref())
	podIDFinder := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref())
	resourceUIDPatternFinder := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref())

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := MustK8sNodeTimeline(ctx, clusterName, componentFieldSet.NodeName)
	componentTimelinePath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, nodeTimelinePath, componentFieldSet.Component)

	cs.AddEvent(componentTimelinePath)

	original := componentFieldSet.Message.Raw()

	foundPods := map[string]struct{}{}
	podFindResults := patternfinder.FindAllWithStarterRunes(original, podIDFinder, false, '"')

	for _, result := range podFindResults {
		podTimelinePath := MustK8sPodTimeline(ctx, clusterName, result.Value.PodNamespace, result.Value.PodName)
		cs.AddEvent(podTimelinePath)
		foundPods[fmt.Sprintf("%s/%s", result.Value.PodNamespace, result.Value.PodName)] = struct{}{}
	}

	containerFindResults := patternfinder.FindAllWithStarterRunes(original, containerIDPatternFinder, false, '"')
	for _, result := range containerFindResults {
		podSandboxID := result.Value.PodSandboxID
		foundPod := patternfinder.FindAllWithStarterRunes(podSandboxID, podIDFinder, true)
		if len(foundPod) == 0 {
			continue
		}
		pod := foundPod[0].Value
		podTimelinePath := MustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName)
		containerTimelinePath := commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, podTimelinePath, result.Value.ContainerName)
		cs.AddEvent(containerTimelinePath)
	}

	resourceFindResults := patternfinder.FindAllWithStarterRunes(original, resourceUIDPatternFinder, false, '"')
	for _, result := range resourceFindResults {
		res := result.Value
		if res.APIVersion == "core/v1" && res.Kind == "pod" {
			if _, ok := foundPods[fmt.Sprintf("%s/%s", res.Namespace, res.Name)]; ok {
				continue
			}
		}
		resTimelinePath := commonlogk8saudit_contract.MustResourceTimeline(ctx, clusterName, res)
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
				containerTimelinePath := commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, podTimelinePath, containerName)
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
	googlecloudlogk8snode_contract.KubeletLogLogToTimelineMapperTaskID,
	&kubeletNodeLogLogToTimelineMapperSetting{},
)
