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

package googlecloudlogk8sevent_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is the task to read the fieldsets required for GKE Event Log parsing.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogk8sevent_contract.FieldSetReaderTaskID,
	googlecloudlogk8sevent_contract.ListLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8sevent_contract.GCPKubernetesEventFieldSetReader{},
		&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	},
)

// KubernetesEventLogIngester handles log ingestion into the KHI v6 builder format.
type KubernetesEventLogIngester struct{}

// RawLogTask returns the task providing the raw logs to ingest.
func (i *KubernetesEventLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *KubernetesEventLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog processes a raw log entry and populates its metadata into LogChangeSet.
func (i *KubernetesEventLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(commonlogk8saudit_contract.LogTypeEvent)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	eventFS, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes event fieldset: %w", err)
	}
	cs.SetSummary(fmt.Sprintf("【%s】%s", eventFS.Reason, eventFS.Message))

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*KubernetesEventLogIngester)(nil)

// LogIngesterTask is the log ingester task for GKE Event Logs.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogk8sevent_contract.LogIngesterTaskID,
	&KubernetesEventLogIngester{},
)

// LogGrouperTask groups logs by the event's resource path so they can be mapped to timelines.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8sevent_contract.LogGrouperTaskID,
	googlecloudlogk8sevent_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		event, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
		if err != nil {
			return "unknown"
		}
		return fmt.Sprintf("cluster=%s,kind=%s,namespace=%s,name=%s", event.ClusterName, event.ResourceKind, event.Namespace, event.Resource)
	},
)

// KubernetesEventTimelineMapper maps grouped GKE Event Logs to resource timelines.
type KubernetesEventTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns the prerequisite log ingester task.
func (m *KubernetesEventTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8sevent_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (m *KubernetesEventTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref(),
	}
}

// GroupedLogTask returns the task providing grouped logs.
func (m *KubernetesEventTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8sevent_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps a single GKE Event Log to its resource timeline path.
func (m *KubernetesEventTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	event, err := log.GetFieldSet(l, &googlecloudlogk8sevent_contract.KubernetesEventFieldSet{})
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to get kubernetes event fieldset: %w", err)
	}

	targetPath := MustResolveK8sResourceTimelinePath(ctx, event)

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(targetPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*KubernetesEventTimelineMapper)(nil)

// LogToTimelineMapperTask is the task to map GKE Event Logs into timeline events.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudlogk8sevent_contract.LogToTimelineMapperTaskID,
	&KubernetesEventTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabel(
		"Kubernetes Event Logs",
		"Gather Kubernetes event logs to visualize cluster events on associated resource timelines.",
		2000,
		true,
	),
)

// MustResolveK8sResourceTimelinePath resolves a KubernetesEventFieldSet to a *khifilev6.TimelinePath.
func MustResolveK8sResourceTimelinePath(ctx context.Context, event *googlecloudlogk8sevent_contract.KubernetesEventFieldSet) *khifilev6.TimelinePath {
	if event.Resource == "" {
		projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, event.ProjectID)
		gkeTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, event.ClusterName)
		return googlecloudlogk8sevent_contract.MustEventExporterTimeline(ctx, gkeTimeline)
	}

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, event.ClusterName)
	apiVersionPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, event.APIVersion)
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionPath, event.ResourceKind)
	if event.Namespace == "cluster-scope" || event.Namespace == "" {
		return commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindPath, event.Resource)
	}
	namespacePath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, event.Namespace)
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespacePath, event.Resource)
}
