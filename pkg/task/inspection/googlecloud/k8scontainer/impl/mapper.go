// Copyright 2026 Google LLC
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

package k8scontainer_impl

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// containerLogIngester implements inspectiontaskbase.LogIngester.
type containerLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *containerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return k8scontainer.ListLogEntriesTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *containerLogIngester) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// ProcessLog is called for each log entry to customize log metadata.
func (i *containerLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(k8scontainer.LogTypeContainer)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil {
		cs.SetSeverity(severity)
	}

	if containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil); err == nil {
		summary := containerFields.Message
		if containerFields.ParsedMessage != nil {
			if sev, err := containerFields.ParsedMessage.Severity(); err == nil {
				cs.SetSeverity(sev)
			}
			if msg, err := containerFields.ParsedMessage.MainMessage(); err == nil && msg != "" {
				summary = msg
			}
		}
		cs.SetSummary(summary)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*containerLogIngester)(nil)

// LogIngesterTask is the task that ingests log metadata into KHI v6 builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	k8scontainer.LogIngesterTaskID,
	&containerLogIngester{},
)

// LogGrouperTask groups logs by associated Pod path.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(k8scontainer.LogGrouperTaskID, k8scontainer.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil)
		if err != nil {
			return "unknown"
		}
		return containerFields.GroupKey()
	})

// containerLogLogToTimelineMapper maps container logs to resource timelines.
type containerLogLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns the task reference of LogIngester.
func (m *containerLogLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8scontainer.LogIngesterTaskID.Ref()
}

// Dependencies returns task dependencies of this mapper.
func (m *containerLogLogToTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scontainer.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides grouped logs.
func (m *containerLogLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8scontainer.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *containerLogLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil)
	if err != nil {
		return nil, struct{}{}, nil
	}

	clusterName := containerFields.ClusterName
	if clusterName == "" || clusterName == "unknown" {
		clusterIdentity := coretask.GetTaskResult(ctx, k8scontainer.ClusterIdentityTaskID.Ref())
		clusterName = clusterIdentity.ClusterName
	}

	clusterPath := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := k8saudit.MustK8sKindTimeline(ctx, apiVersionPath, "pod")
	namespacePath := k8saudit.MustK8sNamespaceTimeline(ctx, kindPath, containerFields.Namespace)
	podPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespacePath, containerFields.PodName)
	containerPath := k8saudit.MustK8sContainerTimeline(
		ctx,
		podPath,
		containerFields.ContainerName,
	)

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(containerPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*containerLogLogToTimelineMapper)(nil)

// LogToTimelineMapperTask creates a task that modifies the KHI v6 TimelineRegistry.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	k8scontainer.LogToTimelineMapperTaskID,
	&containerLogLogToTimelineMapper{},
)

type containerLogPodPhaseTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*containerLogPodPhaseMapperState]
}

func (m *containerLogPodPhaseTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8scontainer.LogIngesterTaskID.Ref()
}

var pathMetadataUID = structured.CompileFieldPath("metadata.uid")

func (m *containerLogPodPhaseTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scontainer.ClusterIdentityTaskID.Ref(),
		k8saudit.TimelineCreationTimeInventoryTaskID.Ref(),
		k8saudit.InitialResourceStateProviderRef,
	}
}

func (m *containerLogPodPhaseTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8scontainer.LogGrouperTaskID.Ref()
}

type containerLogPodPhaseMapperState struct {
	LastNodeName  string
	LastLabels    map[string]string
	AuditLogFound bool
}

func (m *containerLogPodPhaseTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, state *containerLogPodPhaseMapperState) (*khifilev6.TimelineChangeSet, *containerLogPodPhaseMapperState, error) {
	if state != nil && state.AuditLogFound {
		return nil, state, nil
	}

	nodeFields, err := k8scontainer.ExtractGCPContainerLogNodeNameLabel(l.NodeReader)
	if err != nil || nodeFields.NodeName == "" {
		return nil, state, nil
	}

	containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil)
	if err != nil {
		return nil, state, nil
	}

	clusterName := containerFields.ClusterName
	if clusterName == "" || clusterName == "unknown" {
		clusterIdentity := coretask.GetTaskResult(ctx, k8scontainer.ClusterIdentityTaskID.Ref())
		clusterName = clusterIdentity.ClusterName
	}

	podPath, bindingPath := mustPodAndBindingTimelinePaths(ctx, clusterName, containerFields.Namespace, containerFields.PodName)

	nodeNameChanged := state == nil || state.LastNodeName != nodeFields.NodeName
	labelsChanged := state == nil || !maps.Equal(state.LastLabels, nodeFields.PodLabels)
	if !nodeNameChanged && !labelsChanged {
		return nil, state, nil
	}

	// Check if CAI knows about this Pod.
	initialStateProvider := coretask.GetTaskResult(ctx, k8saudit.InitialResourceStateProviderRef)
	initialBody, hasInitialState := initialStateProvider.InitialResourceState(&k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Namespace:  containerFields.Namespace,
		Name:       containerFields.PodName,
	})

	nextState := &containerLogPodPhaseMapperState{
		LastNodeName: nodeFields.NodeName,
		LastLabels:   nodeFields.PodLabels,
	}
	if !nodeNameChanged && hasInitialState {
		return nil, nextState, nil
	}

	var podPhasePath *khifilev6.TimelinePath
	if nodeNameChanged {
		uid := "unknown"
		if hasInitialState {
			if initialUID, err := initialBody.ReadString(pathMetadataUID); err == nil && initialUID != "" {
				uid = initialUID
			}
		}
		podPhasePath = mustPodPhaseTimelinePath(ctx, clusterName, nodeFields.NodeName, containerFields.Namespace, containerFields.PodName, uid)
	}

	// Check if audit log has already recorded creation times for the Pod, its binding, or its phase timeline.
	if state == nil {
		timelineCreationTimes := coretask.GetTaskResult(ctx, k8saudit.TimelineCreationTimeInventoryTaskID.Ref())
		hasPodCreationTime := len(timelineCreationTimes[podPath]) > 0
		hasBindingCreationTime := len(timelineCreationTimes[bindingPath]) > 0
		hasPodPhaseCreationTime := len(timelineCreationTimes[podPhasePath]) > 0

		if hasPodCreationTime || hasBindingCreationTime || hasPodPhaseCreationTime {
			return nil, &containerLogPodPhaseMapperState{AuditLogFound: true}, nil
		}
	}

	podNode, err := makePodManifestNode(containerFields.Namespace, containerFields.PodName, nodeFields.NodeName, nodeFields.PodLabels)
	if err != nil {
		return nil, state, fmt.Errorf("failed to generate pod manifest: %w", err)
	}

	changedTime := time.Unix(0, 0)
	if state != nil {
		changedTime = l.Timestamp
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	if nodeNameChanged {
		cs.AddRevision(podPhasePath, &khifilev6.StagingRevision{
			ChangedTime:  changedTime,
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbUnknown,
			StateType:    k8saudit.RevisionStatePodPhaseUnknown,
		})
	}

	// Only supplement Pod and Binding revisions if CAI does not know about this Pod.
	if !hasInitialState {
		cs.AddRevision(podPath, &khifilev6.StagingRevision{
			ChangedTime:  changedTime,
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbUnknown,
			StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
		})

		if nodeNameChanged {
			bindingNode, err := makeBindingManifestNode(containerFields.Namespace, containerFields.PodName, nodeFields.NodeName)
			if err != nil {
				return nil, state, fmt.Errorf("failed to generate binding manifest: %w", err)
			}
			cs.AddRevision(bindingPath, &khifilev6.StagingRevision{
				ChangedTime:  changedTime,
				ResourceBody: bindingNode,
				Principal:    "N/A",
				VerbType:     k8saudit.VerbUnknown,
				StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
			})
		}
	}

	return cs, nextState, nil
}

// mustPodAndBindingTimelinePaths resolves the timeline paths for a Pod and its binding subresource.
func mustPodAndBindingTimelinePaths(ctx context.Context, clusterName, namespace, podName string) (*khifilev6.TimelinePath, *khifilev6.TimelinePath) {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "pod")
	ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, namespace)
	podPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, podName)
	bindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, podPath, "binding")
	return podPath, bindingPath
}

// makePodManifestNode constructs a structured YAML node for a synthesized Pod manifest.
func makePodManifestNode(namespace, podName, nodeName string, podLabels map[string]string) (structured.Node, error) {
	labels := map[string]any{}
	for k, v := range podLabels {
		labels[k] = v
	}

	podManifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      podName,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]any{
			"nodeName": nodeName,
		},
	}
	return structured.FromGoValue(podManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
}

// makeBindingManifestNode constructs a structured YAML node for a synthesized Binding manifest.
func makeBindingManifestNode(namespace, podName, nodeName string) (structured.Node, error) {
	bindingManifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Binding",
		"metadata": map[string]any{
			"name":      podName,
			"namespace": namespace,
		},
		"target": map[string]any{
			"kind": "Node",
			"name": nodeName,
		},
	}
	return structured.FromGoValue(bindingManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
}

func mustPodPhaseTimelinePath(ctx context.Context, clusterName, nodeName, namespace, podName, uid string) *khifilev6.TimelinePath {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "node")
	nodePath := k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, nodeName)

	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(nodePath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s/%s[%s]", namespace, podName, uid),
		Type: k8saudit.TimelineTypePodPhase,
	})
}

var _ inspectiontaskbase.LogToTimelineMapper[*containerLogPodPhaseMapperState] = (*containerLogPodPhaseTimelineMapper)(nil)

// PodPhaseTimelineMapperTask maps container logs to Pod phase timelines.
var PodPhaseTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*containerLogPodPhaseMapperState](
	k8scontainer.PodPhaseTimelineMapperTaskID,
	&containerLogPodPhaseTimelineMapper{},
)

// TailTask is a nop task that depends on all container log mappers.
var TailTask = coretask.NewTailTask(
	k8scontainer.TailTaskID,
	[]coretask.Dependency{
		k8scontainer.LogToTimelineMapperTaskID.Ref(),
		k8scontainer.PodPhaseTimelineMapperTaskID.Ref(),
		csmcp.LogToTimelineMapperTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel(
		"Kubernetes Container Logs",
		"Gather stdout/stderr logs of containers to visualize application runtime behaviors under associated Pod timelines. Note: The log volume can be very large if the cluster contains many Pods.",
		4000,
		false,
	),
)
