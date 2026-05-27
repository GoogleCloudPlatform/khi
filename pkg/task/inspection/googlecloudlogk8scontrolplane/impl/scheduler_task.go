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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var SchedulerLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogk8scontrolplane_contract.SchedulerLogFilterTaskID,
	googlecloudlogk8scontrolplane_contract.CommonFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
		if err != nil {
			return false
		}
		return componentFieldSet.ComponentParserType() == googlecloudlogk8scontrolplane_contract.ComponentParserTypeScheduler
	},
)

var SchedulerLogFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontrolplane_contract.SchedulerLogFieldSetReaderTaskID,
	googlecloudlogk8scontrolplane_contract.SchedulerLogFilterTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSetReader{},
		&googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSetReader{
			KLogParser: logutil.NewKLogTextParser(false),
		},
	},
)

var SchedulerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.SchedulerLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.SchedulerLogFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

// SchedulerTimelineMapper maps scheduler logs to timeline paths.
type SchedulerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapperV2.
func (m *SchedulerTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (m *SchedulerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.SchedulerLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (m *SchedulerTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapperV2.
func (m *SchedulerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	schedulerMessageFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	compTimeline := MustControlPlaneComponentTimeline(ctx, componentFieldSet.ClusterName, componentFieldSet.ComponentName)
	cs.AddEvent(compTimeline)

	if schedulerMessageFieldSet.HasPodField() {
		clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, componentFieldSet.ClusterName)
		apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
		kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
		namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, schedulerMessageFieldSet.PodNamespace)
		podTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, schedulerMessageFieldSet.PodName)
		cs.AddEvent(podTimeline)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*SchedulerTimelineMapper)(nil)

var SchedulerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(googlecloudlogk8scontrolplane_contract.SchedulerLogToTimelineMapperTaskID, &SchedulerTimelineMapper{})

// MustControlPlaneComponentTimeline returns the timeline path for a Kubernetes control plane component.
func MustControlPlaneComponentTimeline(ctx context.Context, clusterName string, componentName string) *khifilev6.TimelinePath {
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, clusterName)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: componentName,
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})
}
