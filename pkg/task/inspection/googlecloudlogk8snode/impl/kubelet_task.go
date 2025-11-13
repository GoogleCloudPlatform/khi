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

package googlecloudlogk8snode_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

var KubeletLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.KubeletLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Kubelet)

var KubeletLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.KubeletLogGroupTaskID, googlecloudlogk8snode_contract.KubeletLogFilterTaskID.Ref())

var KubeletLogHistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogk8snode_contract.KubeletLogHistoryModifierTaskID, &kubeletNodeLogHistoryModifierSetting{})

type kubeletNodeLogHistoryModifierSetting struct{}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (k *kubeletNodeLogHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (k *kubeletNodeLogHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.KubeletLogGroupTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (k *kubeletNodeLogHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (k *kubeletNodeLogHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	containerdInfo := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID.Ref())

	cs.AddEvent(componentFieldSet.ResourcePath())

	severity, err := componentFieldSet.Message.Severity()
	if err == nil {
		cs.SetLogSeverity(severity)
	}
	summaryReplaceMap := map[string]string{}
	podFindResults := patternfinder.FindAllWithStarterRunes(componentFieldSet.Message.Raw(), containerdInfo.PodSandboxIDInfoFinder, false, '"')

	for _, result := range podFindResults {
		cs.AddEvent(result.Value.ResourcePath())
		summaryReplaceMap[result.Value.PodSandboxID] = toReadablePodSandboxName(result.Value.PodNamespace, result.Value.PodName)
	}

	containerFindResults := patternfinder.FindAllWithStarterRunes(componentFieldSet.Message.Raw(), containerdInfo.ContainerIDInfoFinder, false, '"')
	for _, result := range containerFindResults {
		podSandboxID := result.Value.PodSandboxID
		foundPod := patternfinder.FindAllWithStarterRunes(podSandboxID, containerdInfo.PodSandboxIDInfoFinder, true)
		if len(foundPod) == 0 {
			continue
		}
		pod := foundPod[0].Value
		cs.AddEvent(result.Value.ResourcePath(pod.PodNamespace, pod.PodName))
		summaryReplaceMap[result.Value.ContainerID] = toReadableContainerName(pod.PodNamespace, pod.PodName, result.Value.ContainerName)
	}

	// Kubelet specific severity adjustments
	klogExitCode, err := componentFieldSet.Message.StringField("exitCode")
	if err == nil && klogExitCode != "" && klogExitCode != "0" {
		if klogExitCode == "137" {
			cs.SetLogSeverity(enum.SeverityError)
		} else {
			cs.SetLogSeverity(enum.SeverityWarning)
		}
	}
	summary, err := parseDefaultSummary(componentFieldSet.Message)
	if err != nil {
		summary = componentFieldSet.Message.Raw()
	}
	for k, v := range summaryReplaceMap {
		i := strings.Index(summary, k)
		if i == -1 {
			summary = fmt.Sprintf("%s %s", summary, v)
		} else {
			summary = strings.ReplaceAll(summary, k, v)
		}
	}

	// Kubelet specific resource bindings
	// When this log can't be associated with resource by container id or pod sandbox id, try to get it from klog fields.
	podNameWithNamespace, err := componentFieldSet.Message.StringField("pod")
	if err == nil && podNameWithNamespace != "" {
		podNamespace, podName, err := slashSplittedPodNameToNamespaceAndName(podNameWithNamespace)
		if err == nil {
			containerName, err := componentFieldSet.Message.StringField("containerName")
			if err == nil && containerName != "" {
				cs.AddEvent(resourcepath.Container(podNamespace, podName, containerName))
				cs.SetLogSummary(fmt.Sprintf("%s %s", summary, toReadableContainerName(podNamespace, podName, containerName)))
			} else {
				cs.AddEvent(resourcepath.Pod(podNamespace, podName))
				cs.SetLogSummary(fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(podNamespace, podName)))
			}
		} else {
			cs.SetLogSummary(summary)
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
					cs.AddEvent(resourcepath.Pod(podNamespace, podName))
					summary = fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(podNamespace, podName))
				}
			}
		}
		cs.SetLogSummary(summary)
	}

	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*kubeletNodeLogHistoryModifierSetting)(nil)
