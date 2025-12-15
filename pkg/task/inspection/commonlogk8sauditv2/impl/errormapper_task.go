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

package commonlogk8sauditv2_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// NonSuccessLogLogToTimelineMapperTask is the task to generate history from non-success logs.
var NonSuccessLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](commonlogk8sauditv2_contract.NonSuccessLogLogToTimelineMapperTaskID, &nonSuccessLogLogToTimelineMapperTaskSetting{
	subresourceMapToWriteToParent: map[string]struct{}{
		"status":   {},
		"finalize": {},
		"approve":  {},
	},
})

type nonSuccessLogLogToTimelineMapperTaskSetting struct {
	// subresourceMapToWriteToParent is the map of subresources to write to the parent resource.
	subresourceMapToWriteToParent map[string]struct{}
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return commonlogk8sauditv2_contract.NonSuccessLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	return struct{}{}, e.addEventForLog(l, cs)
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*nonSuccessLogLogToTimelineMapperTaskSetting)(nil)

// addEventForLog adds an event for the log.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) addEventForLog(l *log.Log, cs *history.ChangeSet) error {
	fieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	op := *fieldSet.K8sOperation
	if _, ok := e.subresourceMapToWriteToParent[op.SubResourceName]; op.SubResourceName != "" && ok {
		op.SubResourceName = ""
	}
	cs.AddEvent(resourcepath.ResourcePath{
		Path:               op.ResourcePath(),
		ParentRelationship: enum.RelationshipChild,
	})
	return nil
}
