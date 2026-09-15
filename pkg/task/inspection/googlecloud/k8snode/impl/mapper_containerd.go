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
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"golang.org/x/sync/errgroup"
)

const ContainerdStartingMsg = "starting containerd"
const ContainerdTerminationMsg = "Stop CRI service"

// ContainerdLogFilterTask filters only containerd logs.
var ContainerdLogFilterTask = newParserTypeFilterTask(k8snode.ContainerdLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Containerd)

// ContainerdLogGroupTask groups containerd logs by node and component.
var ContainerdLogGroupTask = newNodeAndComponentNameGrouperTask(k8snode.ContainerdLogGroupTaskID, k8snode.ContainerdLogFilterTaskID.Ref())

// ContainerIDDiscoveryTask discovers mappings between container IDs and GKE pod containers.
var ContainerIDDiscoveryTask = inspectiontaskbase.NewProgressReportableInspectionTask(k8snode.ContainerIDDiscoveryTaskID,
	[]coretask.Dependency{
		k8snode.ContainerdLogFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (k8saudit.ContainerIDToContainerIdentity, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return nil, nil
		}

		logs := coretask.GetTaskResult(ctx, k8snode.ContainerdLogFilterTaskID.Ref())

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

		result := k8saudit.ContainerIDToContainerIdentity{}
		logChan := make(chan *log.Log)
		errGrp, childRoutineCtx := errgroup.WithContext(ctx)
		containerIdentitiesChan := make(chan *k8saudit.ContainerIdentity, runtime.GOMAXPROCS(0))
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			errGrp.Go(func() error {
				for {
					select {
					case <-childRoutineCtx.Done():
						return childRoutineCtx.Err()
					case l, ok := <-logChan:
						if !ok {
							return nil
						}
						processContainerIDDiscoveryForLog(ctx, l, containerIdentitiesChan)
						doneLogCount.Add(1)
					}
				}
			})
		}
		consumerGrp, childConsumerRoutineCtx := errgroup.WithContext(ctx)
		consumerGrp.Go(func() error {
			for {
				select {
				case <-childConsumerRoutineCtx.Done():
					return childConsumerRoutineCtx.Err()
				case c, ok := <-containerIdentitiesChan:
					if !ok {
						return nil
					}
					result[c.ContainerID] = c
				}
			}
		})

		for _, l := range logs {
			logChan <- l
		}
		close(logChan)
		err := errGrp.Wait()
		close(containerIdentitiesChan)
		consumerErr := consumerGrp.Wait()
		if err != nil {
			return nil, err
		}
		if consumerErr != nil {
			return nil, consumerErr
		}

		return result, nil
	},
	coretask.ProvidesTag(k8saudit.TagContainerIDDiscovery),
)

// PodSandboxIDDiscoveryTask discovers mappings between pod sandbox IDs and GKE pods.
var PodSandboxIDDiscoveryTask = inspectiontaskbase.NewProgressReportableInspectionTask(k8snode.PodSandboxIDDiscoveryTaskID,
	[]coretask.Dependency{
		k8snode.ContainerdLogFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return nil, nil
		}
		logs := coretask.GetTaskResult(ctx, k8snode.ContainerdLogFilterTaskID.Ref())

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
		podSandboxIDFinder := patternfinder.NewRadixPatternFinder[*k8snode.PodSandboxIDInfo]()
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

func processPodSandboxIDDiscoveryForLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo]) {
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil || componentFieldSet.Message == nil {
		return
	}
	index, err := findPodSandboxIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	finder.AddPattern(index.PodSandboxID, index)
}

func findPodSandboxIDInfo(jsonPayloadMessage *logutil.ParseStructuredLogResult) (*k8snode.PodSandboxIDInfo, error) {
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
			return &k8snode.PodSandboxIDInfo{
				PodName:      fields["Name"],
				PodNamespace: fields["Namespace"],
				PodSandboxID: sandboxID,
			}, nil
		}
	}
	return nil, fmt.Errorf("pod index information not found:%w", khierrors.ErrNotFound)
}

func processContainerIDDiscoveryForLog(ctx context.Context, l *log.Log, exportTarget chan *k8saudit.ContainerIdentity) {
	componentFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil || componentFieldSet.Message == nil {
		return
	}
	container, err := findContainerIDInfo(componentFieldSet.Message)
	if err != nil {
		return
	}
	exportTarget <- container
}

func findContainerIDInfo(jsonPayloadMessage *logutil.ParseStructuredLogResult) (*k8saudit.ContainerIdentity, error) {
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
			return &k8saudit.ContainerIdentity{
				PodSandboxID:  sandboxID,
				ContainerName: fields["Name"],
				ContainerID:   containerID,
			}, nil
		}
	}
	return nil, fmt.Errorf("container index information not found:%w", khierrors.ErrNotFound)
}

type containerdNodeLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerdNodeLogLogToTimelineMapperSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8snode.ClusterIdentityTaskID.Ref(),
		k8snode.PodSandboxIDDiscoveryTaskID.Ref(),
		k8saudit.ContainerIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerdNodeLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8snode.ContainerdLogGroupTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerdNodeLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8snode.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (c *containerdNodeLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, k8snode.ClusterIdentityTaskID.Ref())
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	podSandboxIDFinder := coretask.GetTaskResult(ctx, k8snode.PodSandboxIDDiscoveryTaskID.Ref())
	containerIDPatternFinder := coretask.GetTaskResult(ctx, k8saudit.ContainerIDPatternFinderTaskID.Ref())
	nodeLogFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := MustK8sNodeTimeline(ctx, clusterName, nodeLogFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, nodeLogFieldSet.Component)

	checkStartingAndTerminationLog(ctx, cs, l, ContainerdStartingMsg, ContainerdTerminationMsg, componentTimelinePath)

	cs.AddEvent(componentTimelinePath)

	raw := nodeLogFieldSet.Message.Raw()
	podFindResults := patternfinder.FindAllWithStarterRunes(raw, podSandboxIDFinder, false, '"', '=')

	for _, result := range podFindResults {
		podTimelinePath := MustK8sPodTimeline(ctx, clusterName, result.Value.PodNamespace, result.Value.PodName)
		cs.AddEvent(podTimelinePath)
	}

	containerFindResults := patternfinder.FindAllWithStarterRunes(raw, containerIDPatternFinder, false, '"', '=')
	for _, result := range containerFindResults {
		podSandboxID := result.Value.PodSandboxID
		foundPod := patternfinder.FindAllWithStarterRunes(podSandboxID, podSandboxIDFinder, true)
		if len(foundPod) == 0 {
			continue
		}
		pod := foundPod[0].Value
		podTimelinePath := MustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName)
		containerTimelinePath := k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, result.Value.ContainerName)
		cs.AddEvent(containerTimelinePath)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*containerdNodeLogLogToTimelineMapperSetting)(nil)

// ContainerdNodeLogLogToTimelineMapperTask registers the mapper for containerd node component logs.
var ContainerdNodeLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	k8snode.ContainerdLogLogToTimelineMapperTaskID,
	&containerdNodeLogLogToTimelineMapperSetting{},
)
