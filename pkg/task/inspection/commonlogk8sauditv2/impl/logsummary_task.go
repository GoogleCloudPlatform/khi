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

// LogSerializerTask is the task to serialize k8s audit logs.
var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(
	commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID,
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

// LogSummaryHistoryModifierTask is the task to generate log summary from given k8s audit log.
var LogSummaryHistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](
	commonlogk8sauditv2_contract.LogSummaryHistoryModifierTaskID,
	&logSummaryHistoryModifierSetting{},
)

type logSummaryHistoryModifierSetting struct{}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (s *logSummaryHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (s *logSummaryHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return commonlogk8sauditv2_contract.LogSummaryGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (s *logSummaryHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (s *logSummaryHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	commonFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})

	if commonFieldSet.IsError {
		cs.SetLogSeverity(enum.SeverityError)
	}

	cs.SetLogSummary(s.logSummary(commonFieldSet))

	return struct{}{}, nil
}

// logSummary generates the summary string from given log field set.
func (s *logSummaryHistoryModifierSetting) logSummary(fieldSet *commonlogk8sauditv2_contract.K8sAuditLogFieldSet) string {
	if fieldSet.IsError {
		return fmt.Sprintf("【%s(%d)】%s %s", fieldSet.StatusMessage, fieldSet.StatusCode, fieldSet.VerbString(), fieldSet.RequestURI)
	} else {
		return fmt.Sprintf("%s %s", fieldSet.VerbString(), fieldSet.RequestURI)
	}
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*logSummaryHistoryModifierSetting)(nil)
