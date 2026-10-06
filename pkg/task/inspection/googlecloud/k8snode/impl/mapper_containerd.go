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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"golang.org/x/sync/errgroup"
)

const containerdStartingMsg = "starting containerd"
const containerdTerminationMsg = "Stop CRI service"

// containerdLogFilterTask filters only containerd logs.
var containerdLogFilterTask = defineParserTypeFilterTask(k8snode.ContainerdLogFilterTaskID, k8snode.ListLogEntriesTaskID.Ref(), k8snode.Containerd)

// containerdLogGroupTask groups containerd logs by node and component.
var containerdLogGroupTask = defineNodeAndComponentNameGrouperTask(k8snode.ContainerdLogGroupTaskID, k8snode.ContainerdLogFilterTaskID.Ref())

// containerIDDiscoveryTask discovers mappings between container IDs and GKE pod containers.
var containerIDDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
	k8snode.ContainerIDDiscoveryTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[k8saudit.ContainerIDToContainerIdentity] {
		containerdLogs := coretask.Use(b, k8snode.ContainerdLogFilterTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.ContainerIDToContainerIdentity, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}

			logs := containerdLogs.Get(ctx)

			tracker := progress.NewTracker(ctx, len(logs), progress.WithUnit("logs"))
			defer tracker.Done()

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
							tracker.Inc()
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
		}
	},
	coretask.ProvidesTag(k8saudit.TagContainerIDDiscovery),
	coretask.WithFeatureGate(k8snode.TailTaskID.Ref()),
)

// podSandboxIDDiscoveryTask discovers mappings between pod sandbox IDs and GKE pods.
var podSandboxIDDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
	k8snode.PodSandboxIDDiscoveryTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo]] {
		containerdLogs := coretask.Use(b, k8snode.ContainerdLogFilterTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			logs := containerdLogs.Get(ctx)

			tracker := progress.NewTracker(ctx, len(logs), progress.WithUnit("logs"))
			defer tracker.Done()

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
							tracker.Inc()
						}
					}
				})
			}

			for _, l := range logs {
				logChan <- l
			}
			close(logChan)
			if err := errGrp.Wait(); err != nil {
				return nil, err
			}

			return podSandboxIDFinder, nil
		}
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

// containerdLogTimelineMapper maps containerd logs to the timelines of their node component and of the pods and containers they mention.
type containerdLogTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity          coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	podSandboxIDFinder       coretask.Input[patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo]]
	containerIDPatternFinder coretask.Input[patternfinder.PatternFinder[*k8saudit.ContainerIdentity]]
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (c *containerdLogTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapContainerdLog(ctx, l, c.clusterIdentity.Get(ctx), c.podSandboxIDFinder.Get(ctx), c.containerIDPatternFinder.Get(ctx))
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*containerdLogTimelineMapper)(nil)

// mapContainerdLog adds events for a containerd log to the timeline of its node component and to the timelines of the pods and containers it mentions.
// It also adds a revision to the component timeline when the log marks the start or the stop of containerd.
func mapContainerdLog(ctx context.Context, l *log.Log, clusterIdentity k8scommon.GoogleCloudClusterIdentity, podSandboxIDFinder patternfinder.PatternFinder[*k8snode.PodSandboxIDInfo], containerIDPatternFinder patternfinder.PatternFinder[*k8saudit.ContainerIdentity]) (*khifilev6.TimelineChangeSet, error) {
	clusterName := clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster)
	nodeLogFieldSet, err := k8snode.ExtractK8sNodeLogCommon(l.NodeReader, nil)
	if err != nil {
		return nil, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	nodeTimelinePath := mustK8sNodeTimeline(ctx, clusterName, nodeLogFieldSet.NodeName)
	componentTimelinePath := k8snode.MustNodeComponentTimeline(ctx, nodeTimelinePath, nodeLogFieldSet.Component)

	checkStartingAndTerminationLog(ctx, cs, l, containerdStartingMsg, containerdTerminationMsg, componentTimelinePath)

	cs.AddEvent(componentTimelinePath)

	raw := nodeLogFieldSet.Message.Raw()
	pods, containerRefs := findPodAndContainerReferences(raw, podSandboxIDFinder, containerIDPatternFinder)
	for _, pod := range pods {
		cs.AddEvent(mustK8sPodTimeline(ctx, clusterName, pod.PodNamespace, pod.PodName))
	}
	for _, containerRef := range containerRefs {
		podTimelinePath := mustK8sPodTimeline(ctx, clusterName, containerRef.Pod.PodNamespace, containerRef.Pod.PodName)
		cs.AddEvent(k8saudit.MustK8sContainerTimeline(ctx, podTimelinePath, containerRef.Container.ContainerName))
	}

	return cs, nil
}

// containerdLogLogToTimelineMapperTask maps containerd logs to timelines.
var containerdLogLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8snode.ContainerdLogLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8snode.LogIngesterTaskID.Ref(),
		GroupedLogs: k8snode.ContainerdLogGroupTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &containerdLogTimelineMapper{
			clusterIdentity:          coretask.Use(b, k8snode.ClusterIdentityTaskID.Ref()),
			podSandboxIDFinder:       coretask.Use(b, k8snode.PodSandboxIDDiscoveryTaskID.Ref()),
			containerIDPatternFinder: coretask.Use(b, k8saudit.ContainerIDPatternFinderTaskID.Ref()),
		}
	},
)
