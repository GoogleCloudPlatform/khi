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
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// K8sNodeLogIngester implements LogIngesterV2 for GKE Node component logs.
type K8sNodeLogIngester struct{}

// RawLogTask returns the raw log provider task.
func (i *K8sNodeLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref()
}

// Dependencies returns the dependencies of the log ingester.
func (i *K8sNodeLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref(),
		commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// ProcessLog populates the LogChangeSet for GKE Node logs.
func (i *K8sNodeLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlogk8snode_contract.LogTypeNode)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	nodeLogFS, err := log.GetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	if err != nil || nodeLogFS.Message == nil {
		return nil, err
	}

	severity, err := nodeLogFS.Message.Severity()
	if err == nil {
		cs.SetSeverity(severity)
	} else {
		cs.SetSeverity(inspectioncore_contract.SeverityInfo)
	}

	raw := nodeLogFS.Message.Raw()
	summaryReplaceMap := map[string]string{}

	podIDFinder := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref())
	if podIDFinder != nil {
		podFindResults := patternfinder.FindAllWithStarterRunes(raw, podIDFinder, false, '"', '=')
		for _, result := range podFindResults {
			summaryReplaceMap[result.Value.PodSandboxID] = toReadablePodSandboxName(result.Value.PodNamespace, result.Value.PodName)
		}
	}

	containerIDPatternFinder := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref())
	if containerIDPatternFinder != nil && podIDFinder != nil {
		containerFindResults := patternfinder.FindAllWithStarterRunes(raw, containerIDPatternFinder, false, '"', '=')
		for _, result := range containerFindResults {
			podSandboxID := result.Value.PodSandboxID
			foundPod := patternfinder.FindAllWithStarterRunes(podSandboxID, podIDFinder, true)
			if len(foundPod) > 0 {
				pod := foundPod[0].Value
				summaryReplaceMap[result.Value.ContainerID] = toReadableContainerName(pod.PodNamespace, pod.PodName, result.Value.ContainerName)
			}
		}
	}

	resourceUIDPatternFinder := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref())
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
				cs.SetSeverity(inspectioncore_contract.SeverityError)
			} else {
				cs.SetSeverity(inspectioncore_contract.SeverityWarning)
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

var _ inspectiontaskbase.LogIngesterV2 = (*K8sNodeLogIngester)(nil)

// LogIngesterTask registers the LogIngesterV2 for GKE Node logs.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudlogk8snode_contract.LogIngesterTaskID,
	&K8sNodeLogIngester{},
)

// CommonFieldSetReaderTask parses the common fieldset used by GKE Node component logs.
var CommonFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID, googlecloudlogk8snode_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSetReader{
		StructuredLogParser: logutil.NewMultiTextLogParser(
			logutil.NewJsonlTextParser(),
			logutil.NewKLogTextParser(true),
			logutil.NewLogfmtTextParser(),
			&logutil.FallbackRawTextLogParser{},
		),
	},
})

// TailTask is a nop task that depends on all node component mappers and other child tasks to group them.
var TailTask = inspectiontaskbase.NewInspectionTask(googlecloudlogk8snode_contract.TailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdLogLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8snode_contract.KubeletLogLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8snode_contract.OtherLogLogToTimelineMapperTaskID.Ref(),

		googlecloudlogk8snode_contract.ContainerIDDiscoveryTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabelV2(
		"Kubernetes Node Logs",
		"Gather logs from Kubernetes node components (e.g., Docker, containerd, or Kubelet) to troubleshoot node-level issues. Note: The log volume can be very large if the cluster contains many nodes.",
		3000,
		false,
	),
)

// newParserTypeFilterTask creates a new filter task that filters only for specific parserType.
func newParserTypeFilterTask(taskid taskid.TaskImplementationID[[]*log.Log], logSource taskid.TaskReference[[]*log.Log], parserType googlecloudlogk8snode_contract.K8sNodeParserType) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewLogFilterTask(
		taskid,
		logSource,
		func(ctx context.Context, l *log.Log) bool {
			componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
			return componentFieldSet.ParserType() == parserType
		},
	)
}

// newNodeAndComponentNameGrouperTask creates a new grouper task with grouping by node name and component name.
func newNodeAndComponentNameGrouperTask(taskid taskid.TaskImplementationID[inspectiontaskbase.LogGroupMap], logSource taskid.TaskReference[[]*log.Log]) coretask.Task[inspectiontaskbase.LogGroupMap] {
	return inspectiontaskbase.NewLogGrouperTask(taskid, logSource, func(ctx context.Context, l *log.Log) string {
		componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
		return fmt.Sprintf("%s-%s", componentFieldSet.NodeName, componentFieldSet.Component)
	})
}
