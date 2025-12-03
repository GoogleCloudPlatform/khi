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
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
)

const ContainerdStartingMsg = "starting containerd"
const ContainerdTerminationMsg = "Stop CRI service"

var ContainerdLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.ContainerdLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Containerd)

var ContainerdLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.ContainerdLogGroupTaskID, googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref())

var ContainerIDDiscoveryTask = commonlogk8sauditv2_contract.ContainerIDInventoryBuilder.DiscoveryTask(googlecloudlogk8snode_contract.ContainerIDDiscoveryTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.ContainerIDToContainerIdentity, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}

		logs := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref())

		doneLogCount := atomic.Int32{}
		updator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := doneLogCount.Load()
			if len(logs) > 0 {
				tp.Percentage = float32(current) / float32(len(logs))
			}
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
		})
		updator.Start(ctx)
		defer updator.Done()

		result := commonlogk8sauditv2_contract.ContainerIDToContainerIdentity{}
		logChan := make(chan *log.Log)
		errGrp, childCtx := errgroup.WithContext(ctx)
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			errGrp.Go(func() error {
				for {
					select {
					case <-childCtx.Done():
						return childCtx.Err()
					case l, ok := <-logChan:
						if !ok {
							return nil
						}
						processContainerIDDiscoveryForLog(ctx, l, result)
						doneLogCount.Add(1)
					}
				}
			})
		}

		for _, l := range logs {
			logChan <- l
		}
		close(logChan)
		errGrp.Wait()

		return result, nil
	},
)

var PodSandboxIDDiscoveryTask = inspectiontaskbase.NewProgressReportableInspectionTask(googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (patternfinder.PatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo], error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}
		logs := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref())

		doneLogCount := atomic.Int32{}
		updator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := doneLogCount.Load()
			if len(logs) > 0 {
				tp.Percentage = float32(current) / float32(len(logs))
			}
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
		})
		updator.Start(ctx)
		defer updator.Done()

		logChan := make(chan *log.Log)
		errGrp, childCtx := errgroup.WithContext(ctx)
		podSandboxIDFinder := patternfinder.NewTriePatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo]()
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			errGrp.Go(func() error {
				for {
					select {
					case <-childCtx.Done():
						return childCtx.Err()
					case l, ok := <-logChan:
						if !ok {
							return nil
						}
						processPodSandboxIDDiscoveryForLog(ctx, l, podSandboxIDFinder)
						doneLogCount.Add(1)
					}
				}
			})
		}

		for _, l := range logs {
			logChan <- l
		}
		close(logChan)
		errGrp.Wait()

		return podSandboxIDFinder, nil
	},
)

func processPodSandboxIDDiscoveryForLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo]) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	index, err := findPodSandboxIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	finder.AddPattern(index.PodSandboxID, index)
}

func findPodSandboxIDInfo(jsonPayloadMessage *logutil.ParseStructuredLogResult) (*googlecloudlogk8snode_contract.PodSandboxIDInfo, error) {
	// RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"
	msg, err := jsonPayloadMessage.MainMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to extract main message: %w", err)
	}
	if strings.HasPrefix(msg, "RunPodSandbox") {
		fields := readGoStructFromString(msg, "PodSandboxMetadata")
		sandboxID := ""
		splitted := strings.Split(msg, "returns sandbox id")
		if len(splitted) >= 2 {
			sandboxID = readNextQuotedString(splitted[1])
		}
		if sandboxID == "" {
			return nil, fmt.Errorf("pod index information not found:%w", khierrors.ErrNotFound)
		}
		if fields["Name"] != "" && fields["Namespace"] != "" {
			return &googlecloudlogk8snode_contract.PodSandboxIDInfo{
				PodName:      fields["Name"],
				PodNamespace: fields["Namespace"],
				PodSandboxID: sandboxID,
			}, nil
		}
	}
	return nil, fmt.Errorf("pod index information not found:%w", khierrors.ErrNotFound)
}

func processContainerIDDiscoveryForLog(ctx context.Context, l *log.Log, exportTarget commonlogk8sauditv2_contract.ContainerIDToContainerIdentity) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	container, err := findContainerIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	exportTarget[container.ContainerID] = container
}

func findContainerIDInfo(jsonPayloadMessage *logutil.ParseStructuredLogResult) (*commonlogk8sauditv2_contract.ContainerIdentity, error) {
	msg, err := jsonPayloadMessage.MainMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to extract main message: %w", err)
	}
	if strings.HasPrefix(msg, "CreateContainer") {
		fields := readGoStructFromString(msg, "ContainerMetadata")
		sandboxID := ""
		splitted := strings.Split(msg, "within sandbox")
		if len(splitted) < 2 {
			return nil, fmt.Errorf("failed to read the sandbox Id from container starting log")
		}
		sandboxID = readNextQuotedString(splitted[1])
		containerID := ""
		splitted = strings.Split(msg, "returns container id")
		if len(splitted) >= 2 {
			containerID = readNextQuotedString(splitted[1])
		}
		if containerID == "" {
			return nil, fmt.Errorf("container index information not found:%w", khierrors.ErrNotFound)
		}
		if fields["Name"] != "" {
			return &commonlogk8sauditv2_contract.ContainerIdentity{
				PodSandboxID:  sandboxID,
				ContainerName: fields["Name"],
				ContainerID:   containerID,
			}, nil
		}
	}
	return nil, fmt.Errorf("container index information not found:%w", khierrors.ErrNotFound)
}

var ContainerdNodeLogHistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogk8snode_contract.ContainerdLogHistoryModifierTaskID, &containerdNodeLogHistoryModifierSetting{})

type containerdNodeLogHistoryModifierSetting struct{}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (c *containerdNodeLogHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.ContainerIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (c *containerdNodeLogHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8snode_contract.ContainerdLogGroupTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (c *containerdNodeLogHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8snode_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (c *containerdNodeLogHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	podSandboxIDFinder := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref())
	containerIDPatternFinder := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ContainerIDPatternFinderTaskID.Ref())
	nodeLogFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})

	checkStartingAndTerminationLog(cs, l, ContainerdStartingMsg, ContainerdTerminationMsg)
	cs.AddEvent(nodeLogFieldSet.ResourcePath())
	msg, err := nodeLogFieldSet.Message.MainMessage()
	if err != nil {
		return struct{}{}, err
	}
	raw := nodeLogFieldSet.Message.Raw()
	summaryReplaceMap := map[string]string{}
	podFindResults := patternfinder.FindAllWithStarterRunes(raw, podSandboxIDFinder, false, '"', '=')

	for _, result := range podFindResults {
		cs.AddEvent(result.Value.ResourcePath())
		summaryReplaceMap[result.Value.PodSandboxID] = toReadablePodSandboxName(result.Value.PodNamespace, result.Value.PodName)
	}

	containerFindResults := patternfinder.FindAllWithStarterRunes(raw, containerIDPatternFinder, false, '"', '=')
	for _, result := range containerFindResults {
		podSandboxID := result.Value.PodSandboxID
		foundPod := patternfinder.FindAllWithStarterRunes(podSandboxID, podSandboxIDFinder, true)
		if len(foundPod) == 0 {
			continue
		}
		pod := foundPod[0].Value
		cs.AddEvent(result.Value.ResourcePath(pod.PodNamespace, pod.PodName))
		summaryReplaceMap[result.Value.ContainerID] = toReadableContainerName(pod.PodNamespace, pod.PodName, result.Value.ContainerName)
	}

	severity, err := nodeLogFieldSet.Message.Severity()
	if err == nil {
		cs.SetLogSeverity(severity)
	}
	summary, err := parseDefaultSummary(nodeLogFieldSet.Message)
	if summary == "" || err != nil {
		summary = msg
	}
	for k, v := range summaryReplaceMap {
		i := strings.Index(summary, k)
		if i == -1 {
			summary = fmt.Sprintf("%s %s", summary, v)
		} else {
			summary = strings.ReplaceAll(summary, k, v)
		}
	}
	cs.SetLogSummary(summary)

	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*containerdNodeLogHistoryModifierSetting)(nil)
