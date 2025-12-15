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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
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
		&googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSetReader{},
	},
)

var SchedulerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.SchedulerLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.SchedulerLogFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

var SchedulerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogk8scontrolplane_contract.SchedulerLogToTimelineMapperTaskID, &schedulerLogToTimelineMapperTaskSetting{})

type schedulerLogToTimelineMapperTaskSetting struct {
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *schedulerLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *schedulerLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.SchedulerLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *schedulerLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *schedulerLogToTimelineMapperTaskSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	commonMainMessage, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	schedulerMessageFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSet{})
	if err != nil {
		return struct{}{}, err
	}

	cs.SetLogSummary(commonMainMessage.Message)
	cs.AddEvent(componentFieldSet.ResourcePath())
	if schedulerMessageFieldSet.HasPodField() {
		cs.AddEvent(schedulerMessageFieldSet.ResourcePath())
	}
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*schedulerLogToTimelineMapperTaskSetting)(nil)
