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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
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

// containerLogIngester implements inspectiontaskbase.LogIngesterV2.
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

var _ inspectiontaskbase.LogIngesterV2 = (*containerLogIngester)(nil)

// LogIngesterTask is the task that ingests log metadata into KHI v6 builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
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

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*containerLogLogToTimelineMapper)(nil)

// LogToTimelineMapperTask creates a task that modifies the KHI v6 TimelineRegistry.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[struct{}](
	googlecloudlogk8scontainer_contract.LogToTimelineMapperTaskID,
	&containerLogLogToTimelineMapper{},
)

type containerLogPodPhaseTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[string]
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

func (m *containerLogPodPhaseTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevNodeName string) (*khifilev6.TimelineChangeSet, string, error) {
	if prevNodeName == "AUDIT" {
		return nil, "AUDIT", nil
	}

	nodeFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{})
	if err != nil || nodeFields.NodeName == "" {
		return nil, prevNodeName, nil
	}

	if prevNodeName == nodeFields.NodeName {
		return nil, prevNodeName, nil
	}

	containerFields, err := log.GetFieldSet(l, &googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{})
	if err != nil {
		return nil, prevNodeName, nil
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
		return nil, "AUDIT", nil
	}

	// Generate Pod phase timeline path under the Node
	podPhasePath := mustPodPhaseTimelinePath(ctx, clusterName, nodeFields.NodeName, containerFields.Namespace, containerFields.PodName, "unknown")

	cs := khifilev6.NewTimelineChangeSet(l)

	cs.AddRevision(podPhasePath, &khifilev6.StagingRevision{
		ChangedTime:  time.Unix(0, 0),
		ResourceBody: nil,
		Principal:    "N/A",
		VerbType:     commonlogk8saudit_contract.VerbUnknown,
		StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
	})

	return cs, nodeFields.NodeName, nil
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

var _ inspectiontaskbase.LogToTimelineMapperV2[string] = (*containerLogPodPhaseTimelineMapper)(nil)

// PodPhaseTimelineMapperTask maps container logs to Pod phase timelines.
var PodPhaseTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[string](
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
	inspectioncore_contract.FeatureTaskLabelV2(
		"Kubernetes Container Logs",
		"Gather stdout/stderr logs of containers to visualize application runtime behaviors under associated Pod timelines. Note: The log volume can be very large if the cluster contains many Pods.",
		4000,
		false,
	),
)
