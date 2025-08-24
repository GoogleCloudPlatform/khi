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

package ossclusterk8s_impl

import (
	"context"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

var NonEventAuditLogFilterTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	ossclusterk8s_contract.NonEventAuditLogFilterTaskID,
	[]taskid.UntypedTaskReference{
		ossclusterk8s_contract.AuditLogFileReaderTaskID.Ref(),
	}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}

		logs := coretask.GetTaskResult(ctx, ossclusterk8s_contract.AuditLogFileReaderTaskID.Ref())

		var auditLogs []*log.Log

		for _, l := range logs {
			verb := l.ReadStringOrDefault("verb", "")
			if l.ReadStringOrDefault("kind", "") == "Event" && l.ReadStringOrDefault("responseObject.kind", "") != "Event" && l.Has("objectRef") {
				if verb == "" || verb == "get" || verb == "watch" || verb == "list" {
					continue
				}
				l.LogType = enum.LogTypeAudit
				auditLogs = append(auditLogs, l)
			}
		}

		return auditLogs, nil
	})
