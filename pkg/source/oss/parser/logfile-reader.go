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

package parser

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectioncontract "github.com/GoogleCloudPlatform/khi/pkg/inspection/contract"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	oss_log "github.com/GoogleCloudPlatform/khi/pkg/source/oss/log"
	oss_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/oss/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var OSSLogFileReader = inspection_task.NewProgressReportableInspectionTask(
	oss_taskid.OSSAPIServerAuditLogFileReader,
	[]taskid.UntypedTaskReference{
		oss_taskid.OSSAPIServerAuditLogFileInputTask.Ref(),
	},
	func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) ([]*log.Log, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		result := task.GetTaskResult(ctx, oss_taskid.OSSAPIServerAuditLogFileInputTask.Ref())

		reader, err := result.GetReader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		logData, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		logLines := strings.Split(string(logData), "\n")
		var logs []*log.Log

		progress.ReportProgressFromArraySync(tp, logLines, func(i int, line string) error {
			if strings.TrimSpace(line) == "" {
				return nil
			}

			l, err := log.NewLogFromYAMLString(line)
			if err != nil {
				return fmt.Errorf("failed to read a log: %w", err)
			}

			err = l.SetFieldSetReader(&oss_log.OSSK8sAuditLogCommonFieldSetReader{})
			if err != nil {
				return err
			}

			// TODO: we may need to consider processing logs not with ResponseComplete stage. All logs not on the ResponseComplete stage will be ignored for now.
			if l.ReadStringOrDefault("stage", "") != "ResponseComplete" {
				return nil
			}

			logs = append(logs, l)
			return nil
		})

		slices.SortFunc(logs, func(a, b *log.Log) int {
			logACommonField := log.MustGetFieldSet(a, &log.CommonFieldSet{})
			logBCommonField := log.MustGetFieldSet(b, &log.CommonFieldSet{})
			return int(logACommonField.Timestamp.UnixNano() - logBCommonField.Timestamp.UnixNano())
		})
		metadataSet := khictx.MustGetValue(ctx, inspectioncontract.InspectionRunMetadata)
		header := typedmap.GetOrDefault(metadataSet, header.HeaderMetadataKey, &header.Header{})

		if len(logs) > 0 {
			startLogCommonField := log.MustGetFieldSet(logs[0], &log.CommonFieldSet{})
			lastLogCommonField := log.MustGetFieldSet(logs[len(logs)-1], &log.CommonFieldSet{})

			header.StartTimeUnixSeconds = startLogCommonField.Timestamp.Unix()
			header.EndTimeUnixSeconds = lastLogCommonField.Timestamp.Unix()
		}

		return logs, nil
	},
)

var OSSEventLogFilter = inspection_task.NewProgressReportableInspectionTask(
	oss_taskid.OSSAPIServerAuditLogFilterNonAuditTaskID,
	[]taskid.UntypedTaskReference{
		oss_taskid.OSSAuditLogFileReader.GetUntypedReference(),
	}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.Log, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		logs := task.GetTaskResult(ctx, oss_taskid.OSSAuditLogFileReader.Ref())

		var eventLogs []*log.Log

		for _, l := range logs {
			if l.ReadStringOrDefault("kind", "") == "Event" && l.ReadStringOrDefault("responseObject.kind", "") == "Event" {
				l.LogType = enum.LogTypeEvent
				eventLogs = append(eventLogs, l)
			}
		}

		return eventLogs, nil
	})

var OSSNonEventLogFilter = inspection_task.NewProgressReportableInspectionTask(
	oss_taskid.OSSAPIServerAuditLogFilterAuditTaskID,
	[]taskid.UntypedTaskReference{
		oss_taskid.OSSAuditLogFileReader.GetUntypedReference(),
	}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.Log, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.Log{}, nil
		}

		logs := task.GetTaskResult(ctx, oss_taskid.OSSAuditLogFileReader.Ref())

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
