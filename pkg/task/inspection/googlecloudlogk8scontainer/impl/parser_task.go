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

package googlecloudlogk8scontainer_impl

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
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask reads fields in parallel.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontainer_contract.FieldSetReaderTaskID, googlecloudlogk8scontainer_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSetReader{},
	&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSetReader{},
})

// containerLogIngester implements inspectiontaskbase.LogIngester.
type containerLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *containerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *containerLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog is called for each log entry to customize log metadata.
func (i *containerLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlogk8scontainer_contract.LogTypeContainer)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{}); err == nil {
		cs.SetSummary(containerFields.Message)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*containerLogIngester)(nil)

// LogIngesterTask is the task that ingests log metadata into KHI v6 builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogk8scontainer_contract.LogIngesterTaskID,
	&containerLogIngester{},
)

// LogGrouperTask groups logs by associated Pod path.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogk8scontainer_contract.LogGrouperTaskID, googlecloudlogk8scontainer_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
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
func (m *containerLogLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns task dependencies of this mapper.
func (m *containerLogLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides grouped logs.
func (m *containerLogLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontainer_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup is called for each log entry to stage mutations via TimelineChangeSet.
func (m *containerLogLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, nil
	}

	clusterName := containerFields.ClusterName
	if clusterName == "" || clusterName == "unknown" {
		clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref())
		clusterName = clusterIdentity.ClusterName
	}

	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionPath, "pod")
	namespacePath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, containerFields.Namespace)
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespacePath, containerFields.PodName)
	containerPath := commonlogk8saudit_contract.MustK8sContainerTimeline(
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
	googlecloudlogk8scontainer_contract.LogToTimelineMapperTaskID,
	&containerLogLogToTimelineMapper{},
)

type containerLogPodPhaseTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*containerLogPodPhaseMapperState]
}

func (m *containerLogPodPhaseTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.LogIngesterTaskID.Ref()
}

func (m *containerLogPodPhaseTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(),
		commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref(),
	}
}

func (m *containerLogPodPhaseTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontainer_contract.LogGrouperTaskID.Ref()
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

	nodeFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{})
	if err != nil || nodeFields.NodeName == "" {
		return nil, state, nil
	}

	containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
	if err != nil {
		return nil, state, nil
	}

	clusterName := containerFields.ClusterName
	if clusterName == "" || clusterName == "unknown" {
		clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref())
		clusterName = clusterIdentity.ClusterName
	}

	// Construct paths for Pod and its binding
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "pod")
	ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, containerFields.Namespace)
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, containerFields.PodName)
	bindingPath := commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, podPath, "binding")

	// Check if audit log has already written to the Pod or its binding timeline
	resourceResult := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref())

	_, hasPodRevision := resourceResult.Revisions[podPath]
	_, hasBindingRevision := resourceResult.Revisions[bindingPath]

	if hasPodRevision || hasBindingRevision {
		return nil, &containerLogPodPhaseMapperState{AuditLogFound: true}, nil
	}

	nodeNameChanged := state == nil || state.LastNodeName != nodeFields.NodeName
	labelsChanged := state == nil || !maps.Equal(state.LastLabels, nodeFields.PodLabels)

	if !nodeNameChanged && !labelsChanged {
		return nil, state, nil
	}

	// Generate Pod phase timeline path under the Node
	podPhasePath := mustPodPhaseTimelinePath(ctx, clusterName, nodeFields.NodeName, containerFields.Namespace, containerFields.PodName, "unknown")

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

	cs := khifilev6.NewTimelineChangeSet(l)

	if nodeNameChanged {
		cs.AddRevision(podPhasePath, &khifilev6.StagingRevision{
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbUnknown,
			StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
		})
	}

	if nodeNameChanged || labelsChanged {
		cs.AddRevision(podPath, &khifilev6.StagingRevision{
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: podNode,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbUnknown,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
		})
	}

	if nodeNameChanged {
		cs.AddRevision(bindingPath, &khifilev6.StagingRevision{
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: bindingNode,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbUnknown,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
		})
	}

	nextState := &containerLogPodPhaseMapperState{
		LastNodeName: nodeFields.NodeName,
		LastLabels:   nodeFields.PodLabels,
	}

	return cs, nextState, nil
}

func mustPodPhaseTimelinePath(ctx context.Context, clusterName, nodeName, namespace, podName, uid string) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "node")
	nodePath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, nodeName)

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nodePath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s/%s[%s]", namespace, podName, uid),
		Type: commonlogk8saudit_contract.TimelineTypePodPhase,
	})
}

var _ inspectiontaskbase.LogToTimelineMapper[*containerLogPodPhaseMapperState] = (*containerLogPodPhaseTimelineMapper)(nil)

// PodPhaseTimelineMapperTask maps container logs to Pod phase timelines.
var PodPhaseTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*containerLogPodPhaseMapperState](
	googlecloudlogk8scontainer_contract.PodPhaseTimelineMapperTaskID,
	&containerLogPodPhaseTimelineMapper{},
)

// TailTask is a nop task that depends on all container log mappers.
var TailTask = inspectiontaskbase.NewInspectionTask(
	googlecloudlogk8scontainer_contract.TailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8scontainer_contract.LogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8scontainer_contract.PodPhaseTimelineMapperTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel(
		"Kubernetes Container Logs",
		"Gather stdout/stderr logs of containers to visualize application runtime behaviors under associated Pod timelines. Note: The log volume can be very large if the cluster contains many Pods.",
		4000,
		false,
	),
)
