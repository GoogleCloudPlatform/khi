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
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

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

// summaryReference represents a resolved pod, container, or resource name for a log ID.
type summaryReference struct {
	readableName string
	skipAppend   bool
}

// processK8sNodeLog sets the log type, timestamp, severity and summary of a Kubernetes node log.
// The summary shows the pods, containers and resources that the finders resolve from the IDs in the log message.
// The summary of a cilium-cni plugin log also shows the subsystem and the Pod written in the log, and shows the Pod only once.
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

	if severity, err := nodeLogFS.Message.Severity(); err == nil {
		cs.SetSeverity(severity)
	} else {
		cs.SetSeverity(inspectioncore.SeverityInfo)
	}

	summary, err := parseDefaultSummary(nodeLogFS.Message)
	if err != nil || summary == "" {
		summary, _ = nodeLogFS.Message.MainMessage()
	}

	if nodeLogFS.Component == "kubelet" {
		applyKubeletExitCodeSeverity(cs, nodeLogFS.Message)
		summary = appendKubeletPodSummary(nodeLogFS.Message, summary)
	}

	summary = formatCiliumCNISummary(nodeLogFS.Message, summary)
	refs := collectSummaryReferences(nodeLogFS.Message, podSandboxIDFinder, containerIDPatternFinder, resourceUIDPatternFinder)
	cs.SetSummary(applySummaryReferences(summary, refs))

	return cs, nil
}

// applyKubeletExitCodeSeverity raises log severity when a kubelet log reports a non-zero exitCode field.
func applyKubeletExitCodeSeverity(cs *khifilev6.LogChangeSet, message *logutil.ParseStructuredLogResult) {
	exitCode, _ := message.StringField("exitCode")
	if exitCode == "" || exitCode == "0" {
		return
	}
	if exitCode == "137" {
		cs.SetSeverity(inspectioncore.SeverityError)
	} else {
		cs.SetSeverity(inspectioncore.SeverityWarning)
	}
}

// appendKubeletPodSummary appends readable pod or container names parsed from kubelet klog fields to summary.
func appendKubeletPodSummary(message *logutil.ParseStructuredLogResult, summary string) string {
	if podRef, _ := message.StringField("pod"); podRef != "" {
		ns, name, err := slashSplittedPodNameToNamespaceAndName(podRef)
		if err != nil {
			return summary
		}
		if container, _ := message.StringField("containerName"); container != "" {
			return fmt.Sprintf("%s %s", summary, toReadableContainerName(ns, name, container))
		}
		return fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(ns, name))
	}

	podNames, _ := message.StringField("pods")
	for _, rawPod := range strings.Split(strings.Trim(podNames, "[]"), ",") {
		if ns, name, err := slashSplittedPodNameToNamespaceAndName(strings.Trim(rawPod, `"`)); err == nil {
			summary = fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(ns, name))
		}
	}
	return summary
}

// formatCiliumCNISummary prepends the cilium-cni tag and appends the referenced Pod name when present.
func formatCiliumCNISummary(message *logutil.ParseStructuredLogResult, summary string) string {
	if !isCiliumCNILog(message) {
		return summary
	}
	// The plugin can end its messages with a space, which would otherwise widen the gap before the appended references.
	summary = fmt.Sprintf("[cilium-cni] %s", strings.TrimSpace(summary))
	if ns, name, found := findCiliumCNIPodReference(message); found {
		summary = fmt.Sprintf("%s %s", summary, toReadablePodSandboxName(ns, name))
	}
	return summary
}

// collectSummaryReferences maps pod sandbox IDs, container IDs, and resource UIDs in message to readable summary replacements.
func collectSummaryReferences(
	message *logutil.ParseStructuredLogResult,
	podSandboxIDFinder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo],
	containerIDPatternFinder patternfinder.PatternFinder[*k8saudit.ContainerIdentity],
	resourceUIDPatternFinder patternfinder.PatternFinder[*k8saudit.ResourceIdentity],
) map[string]summaryReference {
	raw := message.Raw()
	refs := map[string]summaryReference{}

	ciliumPodNs, ciliumPodName, hasCiliumPod := findCiliumCNIPodReference(message)
	isCiliumPod := func(namespace, name string) bool {
		return hasCiliumPod && namespace == ciliumPodNs && name == ciliumPodName
	}

	pods, containerRefs := findPodAndContainerReferences(raw, podSandboxIDFinder, containerIDPatternFinder)
	for _, pod := range pods {
		refs[pod.PodSandboxID] = summaryReference{
			readableName: toReadablePodSandboxName(pod.PodNamespace, pod.PodName),
			skipAppend:   isCiliumPod(pod.PodNamespace, pod.PodName),
		}
	}
	for _, containerRef := range containerRefs {
		refs[containerRef.Container.ContainerID] = summaryReference{
			readableName: toReadableContainerName(containerRef.Pod.PodNamespace, containerRef.Pod.PodName, containerRef.Container.ContainerName),
		}
	}

	if resourceUIDPatternFinder != nil {
		for _, result := range patternfinder.FindAllWithStarterRunes(raw, resourceUIDPatternFinder, false, nodeLogUIDStarterRunes...) {
			if uid, err := result.GetMatchedString(raw); err == nil {
				refs[uid] = summaryReference{
					readableName: toReadableResourceName(result.Value.APIVersion, result.Value.Kind, result.Value.Namespace, result.Value.Name),
					skipAppend:   result.Value.Kind == "pod" && isCiliumPod(result.Value.Namespace, result.Value.Name),
				}
			}
		}
	}
	return refs
}

// applySummaryReferences replaces IDs present in summary with their readable names and appends unmentioned references.
func applySummaryReferences(summary string, refs map[string]summaryReference) string {
	for id, ref := range refs {
		if strings.Contains(summary, id) {
			summary = strings.ReplaceAll(summary, id, ref.readableName)
			continue
		}
		if !ref.skipAppend {
			summary = fmt.Sprintf("%s %s", summary, ref.readableName)
		}
	}
	return summary
}

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
