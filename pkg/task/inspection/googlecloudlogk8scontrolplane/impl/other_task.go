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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

var OtherLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogk8scontrolplane_contract.OtherLogFilterTaskID,
	googlecloudlogk8scontrolplane_contract.CommonFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
		if err != nil {
			return false
		}
		return componentFieldSet.ComponentParserType() == googlecloudlogk8scontrolplane_contract.ComponentParserTypeOther
	},
)

var OtherLogFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontrolplane_contract.OtherLogFieldSetReaderTaskID,
	googlecloudlogk8scontrolplane_contract.OtherLogFilterTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSetReader{},
	},
)

var OtherGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.OtherLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.OtherLogFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
		if err != nil {
			return ""
		}
		return componentFieldSet.ComponentName
	},
)

// OtherTimelineMapper maps other control plane logs to timeline paths.
type OtherTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.OtherLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *OtherTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, componentFieldSet.ProjectID)
	gkeTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, componentFieldSet.ClusterName)
	compTimeline := googlecloudlogk8scontrolplane_contract.MustControlPlaneComponentTimeline(ctx, gkeTimeline, componentFieldSet.ComponentName)
	cs.AddEvent(compTimeline)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*OtherTimelineMapper)(nil)

var OtherLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogk8scontrolplane_contract.OtherLogToTimelineMapperTaskID, &OtherTimelineMapper{})
