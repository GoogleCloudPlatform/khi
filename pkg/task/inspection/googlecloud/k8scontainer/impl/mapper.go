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
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// processContainerLog sets the log type, timestamp, severity and summary of a Kubernetes container log.
func processContainerLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
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

// logIngesterTask is the task that ingests log metadata into KHI v6 builder.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	k8scontainer.LogIngesterTaskID,
	k8scontainer.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processContainerLog
	},
)

// logGrouperTask groups logs by associated Pod path.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	k8scontainer.LogGrouperTaskID,
	k8scontainer.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil)
			if err != nil {
				return "unknown"
			}
			return containerFields.GroupKey()
		}
	},
)

// containerLogTimelineMapper maps container logs to resource timelines.
type containerLogTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *containerLogTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapContainerLog(ctx, l, m.clusterIdentity.Get(ctx).ClusterName)
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*containerLogTimelineMapper)(nil)

// mapContainerLog adds an event for a Kubernetes container log to the timeline of its container.
// It uses fallbackClusterName when the cluster name in the log is empty or "unknown".
func mapContainerLog(ctx context.Context, l *log.Log, fallbackClusterName string) (*khifilev6.TimelineChangeSet, error) {
	containerFields, err := k8scontainer.ExtractK8sContainerLog(l.NodeReader, nil)
	if err != nil {
		return nil, nil
	}

	clusterName := containerFields.ClusterName
	if clusterName == "" || clusterName == "unknown" {
		clusterName = fallbackClusterName
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

	return cs, nil
}

// logToTimelineMapperTask creates a task that modifies the KHI v6 TimelineRegistry.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8scontainer.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8scontainer.LogIngesterTaskID.Ref(),
		GroupedLogs: k8scontainer.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &containerLogTimelineMapper{
			clusterIdentity: coretask.Use(b, k8scontainer.ClusterIdentityTaskID.Ref()),
		}
	},
)

var pathMetadataUID = structured.CompileFieldPath("metadata.uid")

type containerLogPodPhaseMapperState struct {
	LastNodeName  string
	LastLabels    map[string]string
	AuditLogFound bool
}

// containerLogPodPhaseTimelineMapper infers Pod phase, Pod and binding revisions from the node names and labels in container logs when no audit log recorded them.
type containerLogPodPhaseTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*containerLogPodPhaseMapperState]
	clusterIdentity      coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	initialStateProvider coretask.Input[k8saudit.InitialResourceStateProvider]
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (m *containerLogPodPhaseTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, state *containerLogPodPhaseMapperState) (*khifilev6.TimelineChangeSet, *containerLogPodPhaseMapperState, error) {
	return mapContainerLogPodPhase(ctx, l, state, m.clusterIdentity.Get(ctx).ClusterName, m.initialStateProvider.Get(ctx))
}

var _ inspectiontaskbase.TimelineMapper[*containerLogPodPhaseMapperState] = (*containerLogPodPhaseTimelineMapper)(nil)

// mapContainerLogPodPhase supplements the Pod phase, Pod and binding revisions of the Pod that a container log belongs to from the node name and labels in the log.
// It skips Pods whose revisions audit logs already recorded, adds Pod and binding revisions only when initialStateProvider does not know the Pod,
// and uses fallbackClusterName when the cluster name in the log is empty or "unknown".
func mapContainerLogPodPhase(ctx context.Context, l *log.Log, state *containerLogPodPhaseMapperState, fallbackClusterName string, initialStateProvider k8saudit.InitialResourceStateProvider) (*khifilev6.TimelineChangeSet, *containerLogPodPhaseMapperState, error) {
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
		clusterName = fallbackClusterName
	}

	// Construct paths for Pod and its binding
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "pod")
	ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, containerFields.Namespace)
	podPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, containerFields.PodName)
	bindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, podPath, "binding")

	nodeNameChanged := state == nil || state.LastNodeName != nodeFields.NodeName
	labelsChanged := state == nil || !maps.Equal(state.LastLabels, nodeFields.PodLabels)
	if !nodeNameChanged && !labelsChanged {
		return nil, state, nil
	}

	// Check if CAI knows about this Pod.
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

	// Check if audit log has already written to the Pod, its binding, or its phase timeline.
	// When CAI knows about the Pod, a revision on podPath may originate from CAI rather than audit logs.
	if state == nil {
		builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
		hasBindingRevision := builder.TimelineAccumulator.HasRevision(bindingPath)
		hasPodPhaseRevision := builder.TimelineAccumulator.HasRevision(podPhasePath)
		hasAuditPodRevision := !hasInitialState && builder.TimelineAccumulator.HasRevision(podPath)

		if hasBindingRevision || hasPodPhaseRevision || hasAuditPodRevision {
			return nil, &containerLogPodPhaseMapperState{AuditLogFound: true}, nil
		}
	}

	labels := map[string]any{}
	for k, v := range nodeFields.PodLabels {
		labels[k] = v
	}

	podManifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      containerFields.PodName,
			"namespace": containerFields.Namespace,
			"labels":    labels,
		},
		"spec": map[string]any{
			"nodeName": nodeFields.NodeName,
		},
	}
	podNode, err := structured.FromGoValue(podManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		return nil, state, fmt.Errorf("failed to generate pod manifest: %w", err)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	if nodeNameChanged {
		cs.AddRevision(podPhasePath, &khifilev6.StagingRevision{
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbUnknown,
			StateType:    k8saudit.RevisionStatePodPhaseUnknown,
		})
	}

	// Only supplement Pod and Binding revisions if CAI does not know about this Pod.
	if !hasInitialState {
		cs.AddRevision(podPath, &khifilev6.StagingRevision{
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbUnknown,
			StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
		})

		if nodeNameChanged {
			bindingManifest := map[string]any{
				"apiVersion": "v1",
				"kind":       "Binding",
				"metadata": map[string]any{
					"name":      containerFields.PodName,
					"namespace": containerFields.Namespace,
				},
				"target": map[string]any{
					"kind": "Node",
					"name": nodeFields.NodeName,
				},
			}
			bindingNode, err := structured.FromGoValue(bindingManifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				return nil, state, fmt.Errorf("failed to generate binding manifest: %w", err)
			}
			cs.AddRevision(bindingPath, &khifilev6.StagingRevision{
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: bindingNode,
				Principal:    "N/A",
				VerbType:     k8saudit.VerbUnknown,
				StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
			})
		}
	}

	return cs, nextState, nil
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

// podPhaseTimelineMapperTask maps container logs to Pod phase timelines.
var podPhaseTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8scontainer.PodPhaseTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8scontainer.LogIngesterTaskID.Ref(),
		GroupedLogs: k8scontainer.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*containerLogPodPhaseMapperState] {
		// The audit log mappers must write their revisions first because this mapper skips Pods that already have them.
		coretask.After(b, k8saudit.ResourceRevisionLogToTimelineMapperTaskID.Ref())
		coretask.After(b, k8saudit.PodPhaseLogToTimelineMapperTaskID.Ref())
		return &containerLogPodPhaseTimelineMapper{
			clusterIdentity:      coretask.Use(b, k8scontainer.ClusterIdentityTaskID.Ref()),
			initialStateProvider: coretask.Use(b, k8saudit.InitialResourceStateProviderRef),
		}
	},
)

// tailTask is a nop task that depends on all container log mappers.
var tailTask = coretask.DefineTailTask(
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
