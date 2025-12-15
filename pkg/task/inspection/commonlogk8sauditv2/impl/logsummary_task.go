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
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// LogIngesterTask is the task to serialize k8s audit logs.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	commonlogk8sauditv2_contract.K8sAuditLogIngesterTaskID,
	commonlogk8sauditv2_contract.K8sAuditLogProviderRef,
)

// LogSummaryGrouperTask is the task to group logs for summary generation.
var LogSummaryGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	commonlogk8sauditv2_contract.LogSummaryGrouperTaskID,
	commonlogk8sauditv2_contract.K8sAuditLogProviderRef,
	func(ctx context.Context, l *log.Log) string {
		commonFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
		return commonFieldSet.K8sOperation.ResourcePath()
	},
)

// LogSummaryLogToTimelineMapperTask is the task to generate log summary from given k8s audit log.
var LogSummaryLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	commonlogk8sauditv2_contract.LogSummaryLogToTimelineMapperTaskID,
	&logSummaryLogToTimelineMapperSetting{},
)

type logSummaryLogToTimelineMapperSetting struct{}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (s *logSummaryLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (s *logSummaryLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return commonlogk8sauditv2_contract.LogSummaryGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (s *logSummaryLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (s *logSummaryLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	commonFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})

	if commonFieldSet.IsError {
		cs.SetLogSeverity(enum.SeverityError)
	}

	cs.SetLogSummary(s.logSummary(commonFieldSet))

	return struct{}{}, nil
}

// logSummary generates the summary string from given log field set.
func (s *logSummaryLogToTimelineMapperSetting) logSummary(fieldSet *commonlogk8sauditv2_contract.K8sAuditLogFieldSet) string {
	if fieldSet.IsError {
		return fmt.Sprintf("【%s(%d)】%s %s", fieldSet.StatusMessage, fieldSet.StatusCode, fieldSet.VerbString(), fieldSet.RequestURI)
	} else {
		return fmt.Sprintf("%s %s", fieldSet.VerbString(), fieldSet.RequestURI)
	}
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*logSummaryLogToTimelineMapperSetting)(nil)
