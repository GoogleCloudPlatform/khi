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
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
)

const ContainerdStartingMsg = "starting containerd"
const ContainerdTerminationMsg = "Stop CRI service"

var ContainerdLogFilterTask = newParserTypeFilterTask(googlecloudlogk8snode_contract.ContainerdLogFilterTaskID, googlecloudlogk8snode_contract.CommonFieldsetReaderTaskID.Ref(), googlecloudlogk8snode_contract.Containerd)

var ContainerdLogGroupTask = newNodeAndComponentNameGrouperTask(googlecloudlogk8snode_contract.ContainerdLogGroupTaskID, googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref())

var ContainerdIDDiscoveryTask = inspectiontaskbase.NewProgressReportableInspectionTask(googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (*googlecloudlogk8snode_contract.ContainerdRelationshipRegistry, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}
		relationshipRepository := googlecloudlogk8snode_contract.NewContainerdRelationshipRegistry()
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
						processPodSandboxIDDiscoveryForLog(ctx, l, relationshipRepository)
						processContainerIDDiscoveryForLog(ctx, l, relationshipRepository)
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

		return relationshipRepository, nil
	},
)

func processPodSandboxIDDiscoveryForLog(ctx context.Context, l *log.Log, relationshipRepository *googlecloudlogk8snode_contract.ContainerdRelationshipRegistry) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	index, err := findPodSandboxIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	relationshipRepository.PodSandboxIDInfoFinder.AddPattern(index.PodSandboxID, index)
}

func findPodSandboxIDInfo(jsonPayloadMessage string) (*googlecloudlogk8snode_contract.PodSandboxIDInfo, error) {
	// RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"
	msg, err := logutil.ExtractKLogField(jsonPayloadMessage, "msg")
	if err != nil {
		return nil, fmt.Errorf("failed to extract main message: %w", err)
	}
	if msg == "" {
		return nil, fmt.Errorf("main message not found in log")
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

func processContainerIDDiscoveryForLog(ctx context.Context, l *log.Log, relationshipRepository *googlecloudlogk8snode_contract.ContainerdRelationshipRegistry) {
	componentFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})
	index, err := findContainerIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	relationshipRepository.ContainerIDInfoFinder.AddPattern(index.ContainerID, index)
}

func findContainerIDInfo(jsonPayloadMessage string) (*googlecloudlogk8snode_contract.ContainerIDInfo, error) {
	msg, err := logutil.ExtractKLogField(jsonPayloadMessage, "msg")
	if err != nil {
		return nil, fmt.Errorf("failed to extract main message: %w", err)
	}
	if msg == "" {
		return nil, fmt.Errorf("main message not found in log")
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
			return &googlecloudlogk8snode_contract.ContainerIDInfo{
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
		googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID.Ref(),
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
	containerdInfo := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID.Ref())
	nodeLogFieldSet := log.MustGetFieldSet(l, &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{})

	checkStartingAndTerminationLog(cs, l, ContainerdStartingMsg, ContainerdTerminationMsg)
	cs.AddEvent(nodeLogFieldSet.ResourcePath())
	msg, err := logutil.ExtractKLogField(nodeLogFieldSet.Message, "msg")
	if msg == "" || err != nil {
		return struct{}{}, err
	}
	summaryReplaceMap := map[string]string{}
	podFindResults := patternfinder.FindAllWithStarterRunes(nodeLogFieldSet.Message, containerdInfo.PodSandboxIDInfoFinder, false, '"', '=')

	for _, result := range podFindResults {
		cs.AddEvent(result.Value.ResourcePath())
		summaryReplaceMap[result.Value.PodSandboxID] = toReadablePodSandboxName(result.Value.PodNamespace, result.Value.PodName)
	}

	containerFindResults := patternfinder.FindAllWithStarterRunes(nodeLogFieldSet.Message, containerdInfo.ContainerIDInfoFinder, false, '"', '=')
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

	severity := logutil.ExractKLogSeverity(nodeLogFieldSet.Message)
	cs.SetLogSeverity(severity)
	summary, err := parseDefaultSummary(nodeLogFieldSet.Message)
	if summary == "" || err != nil {
		summary = nodeLogFieldSet.Message
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
