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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// processK8sNodeLog sets the log type, timestamp, severity and summary of a Kubernetes node log.
// The summary shows the pods, containers and resources that the finders resolve from the IDs in the log message.
func processK8sNodeLog(ctx context.Context, l *log.Log, podSandboxIDFinder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], containerIDPatternFinder patternfinder.PatternFinder[*k8saudit.ContainerIdentity], resourceUIDPatternFinder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(k8snode.LogTypeNode)
	cs.SetTimestamp(l.Timestamp)

	nodeLogFS, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil || nodeLogFS.Message == nil {
		return nil, err
	}

	severity, err := nodeLogFS.Message.Severity()
	if err == nil {
		cs.SetSeverity(severity)
	} else {
		cs.SetSeverity(inspectioncore.SeverityInfo)
	}

	raw := nodeLogFS.Message.Raw()
	summaryReplaceMap := map[string]string{}

	pods, containerRefs := findPodAndContainerReferences(raw, podSandboxIDFinder, containerIDPatternFinder)
	for _, pod := range pods {
		summaryReplaceMap[pod.PodSandboxID] = toReadablePodSandboxName(pod.PodNamespace, pod.PodName)
	}
	for _, containerRef := range containerRefs {
		summaryReplaceMap[containerRef.Container.ContainerID] = toReadableContainerName(containerRef.Pod.PodNamespace, containerRef.Pod.PodName, containerRef.Container.ContainerName)
	}

	if resourceUIDPatternFinder != nil {
		resourceFindResults := patternfinder.FindAllWithStarterRunes(raw, resourceUIDPatternFinder, false, '"', '=')
		for _, result := range resourceFindResults {
			uid, err := result.GetMatchedString(raw)
			if err == nil {
				summaryReplaceMap[uid] = toReadableResourceName(result.Value.APIVersion, result.Value.Kind, result.Value.Namespace, result.Value.Name)
			}
		}
	}

	summary, err := parseDefaultSummary(nodeLogFS.Message)
	if err != nil || summary == "" {
		summary, _ = nodeLogFS.Message.MainMessage()
	}

	if nodeLogFS.Component == "kubelet" {
		klogExitCode, err := nodeLogFS.Message.StringField("exitCode")
		if err == nil && klogExitCode != "" && klogExitCode != "0" {
			if klogExitCode == "137" {
				cs.SetSeverity(inspectioncore.SeverityError)
			} else {
				cs.SetSeverity(inspectioncore.SeverityWarning)
			}
		}

		podNameWithNamespace, err := nodeLogFS.Message.StringField("pod")
		if err == nil && podNameWithNamespace != "" {
			podNamespace, podName, err := slashSplittedPodNameToNamespaceAndName(podNameWithNamespace)
			if err == nil {
				containerName, err := nodeLogFS.Message.StringField("containerName")
				if err == nil && containerName != "" {
					summary = fmt.Sprintf("%s %s", summary, toReadableContainerName(podNamespace, podName, containerName))
				} else {
					summary = fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(podNamespace, podName))
				}
			}
		} else {
			podNames, err := nodeLogFS.Message.StringField("pods")
			if err == nil && podNames != "" {
				podNames = strings.Trim(podNames, "[]")
				podNamesSplitted := strings.Split(podNames, ",")
				for _, podNamespaceAndNameWithSlash := range podNamesSplitted {
					podNamespaceAndNameWithSlash = strings.Trim(podNamespaceAndNameWithSlash, `"`)
					podNamespace, podName, err := slashSplittedPodNameToNamespaceAndName(podNamespaceAndNameWithSlash)
					if err == nil {
						summary = fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(podNamespace, podName))
					}
				}
			}
		}
	}

	for k, v := range summaryReplaceMap {
		i := strings.Index(summary, k)
		if i == -1 {
			summary = fmt.Sprintf("%s %s", summary, v)
		} else {
			summary = strings.ReplaceAll(summary, k, v)
		}
	}
	cs.SetSummary(summary)

	return cs, nil
}

// logIngesterTask ingests the metadata of Kubernetes node logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	k8snode.LogIngesterTaskID,
	k8snode.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		podSandboxIDFinder := coretask.Use(b, k8snode.PodSandboxIDDiscoveryTaskID.Ref())
		containerIDPatternFinder := coretask.Use(b, k8saudit.ContainerIDPatternFinderTaskID.Ref())
		resourceUIDPatternFinder := coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
		return func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
			return processK8sNodeLog(ctx, l, podSandboxIDFinder.Get(ctx), containerIDPatternFinder.Get(ctx), resourceUIDPatternFinder.Get(ctx))
		}
	},
)

// tailTask is the feature task of Kubernetes node logs that completes after all node component log mappers.
var tailTask = coretask.DefineTailTask(
	k8snode.TailTaskID,
	[]coretask.Dependency{
		k8snode.ContainerdLogLogToTimelineMapperTaskID.Ref(),
		k8snode.KubeletLogLogToTimelineMapperTaskID.Ref(),
		k8snode.OtherLogLogToTimelineMapperTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel(
		"Kubernetes Node Logs",
		"Gather logs from Kubernetes node components (e.g., Docker, containerd, or Kubelet) to troubleshoot node-level issues. Note: The log volume can be very large if the cluster contains many nodes.",
		3000,
		false,
	),
)

// defineParserTypeFilterTask defines a filter task that passes only the logs of the given parser type.
func defineParserTypeFilterTask(taskID taskid.TaskImplementationID[[]*log.Log], logSource taskid.TaskReference[[]*log.Log], parserType k8snode.K8sNodeParserType) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.DefineLogFilterTask(taskID, logSource, func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			gotParserType, err := k8snode.ExtractK8sNodeParserType(l.NodeReader)
			if err != nil {
				return false
			}
			return gotParserType == parserType
		}
	})
}

// defineNodeAndComponentNameGrouperTask defines a grouper task that groups logs by node name and component name.
func defineNodeAndComponentNameGrouperTask(taskID taskid.TaskImplementationID[inspectiontaskbase.LogGroupMap], logSource taskid.TaskReference[[]*log.Log]) coretask.Task[inspectiontaskbase.LogGroupMap] {
	return inspectiontaskbase.DefineLogGrouperTask(taskID, logSource, func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
			if err != nil {
				return ""
			}
			return fmt.Sprintf("%s-%s", componentFieldSet.NodeName, componentFieldSet.Component)
		}
	})
}
