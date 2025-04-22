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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	oss_log "github.com/GoogleCloudPlatform/khi/pkg/source/oss/log"
	oss_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/oss/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var OSSLogFileReader = inspection_task.NewProgressReportableInspectionTask(
	oss_taskid.OSSAPIServerAuditLogFileReader,
	[]taskid.UntypedTaskReference{
		oss_taskid.OSSAPIServerAuditLogFileInputTask.GetTaskReference(),
		inspection_task.ReaderFactoryGeneratorTaskID.GetTaskReference(),
	},
	func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}
		result := task.GetTaskResult(ctx, oss_taskid.OSSAPIServerAuditLogFileInputTask.GetTaskReference())
		readerFactory := task.GetTaskResult(ctx, inspection_task.ReaderFactoryGeneratorTaskID.GetTaskReference())

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
		var logs []*log.LogEntity

		for _, line := range logLines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to parse JSON line: %v", err))
				continue
			}

			yamlData, err := yaml.Marshal(jsonData)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to convert to YAML: %v", err))
				continue
			}

			logReader, err := readerFactory.NewReader(adapter.Yaml(string(yamlData)))
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to parse YAML as the structure data: %v", err))
				continue
			}
			log := log.NewLogEntity(logReader, &oss_log.OSSAuditLogFieldExtractor{})

			// TODO: we may need to consider processing logs not with ResponseComplete stage. All logs not on the ResponseComplete stage will be ignored for now.
			if log.GetStringOrDefault("stage", "") != "ResponseComplete" {
				continue
			}

			logs = append(logs, log)
		}
		slices.SortFunc(logs, func(a, b *log.LogEntity) int { return int(a.Timestamp().UnixNano() - b.Timestamp().UnixNano()) })
		metadataSet := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
		header := typedmap.GetOrDefault(metadataSet, header.HeaderMetadataKey, &header.Header{})

		if len(logs) > 0 {
			header.StartTimeUnixSeconds = logs[0].Timestamp().Unix()
			header.EndTimeUnixSeconds = logs[len(logs)-1].Timestamp().Unix()
		}

		return logs, nil
	},
)

var OSSEventLogFilter = inspection_task.NewProgressReportableInspectionTask(
	oss_taskid.OSSAPIServerAuditLogFilterNonAuditTaskID,
	[]taskid.UntypedTaskReference{
		oss_taskid.OSSAuditLogFileReader.GetUntypedReference(),
	}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}
		logs := task.GetTaskResult(ctx, oss_taskid.OSSAuditLogFileReader.GetTaskReference())

		var eventLogs []*log.LogEntity

		for _, l := range logs {
			if l.GetStringOrDefault("kind", "") == "Event" && l.GetStringOrDefault("responseObject.kind", "") == "Event" {
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
	}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
		if taskMode == inspection_task_interface.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}

		logs := task.GetTaskResult(ctx, oss_taskid.OSSAuditLogFileReader.GetTaskReference())

		var auditLogs []*log.LogEntity

		for _, l := range logs {
			verb := l.GetStringOrDefault("verb", "")
			if l.GetStringOrDefault("kind", "") == "Event" && l.GetStringOrDefault("responseObject.kind", "") != "Event" && l.Fields.Has("objectRef") {
				if verb == "" || verb == "get" || verb == "watch" || verb == "list" {
					continue
				}
				l.LogType = enum.LogTypeAudit
				auditLogs = append(auditLogs, l)
			}
		}

		return auditLogs, nil
	})
