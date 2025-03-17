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
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/constant"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/form"
	oss_log "github.com/GoogleCloudPlatform/khi/pkg/source/oss/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var OSSAuditLogFileReaderTaskID = constant.OSSTaskPrefix + "audit-logs-reader"
var OSSAuditLogEventLogsFilterTaskID = constant.OSSTaskPrefix + "event-logs-filter"
var OSSAuditLogNonEventFilterTaskID = constant.OSSTaskPrefix + "non-event-logs-filter"

var OSSLogFileReader = inspection_task.NewInspectionProcessor(
	OSSAuditLogFileReaderTaskID,
	[]string{
		form.AuditLogFilesForm.ID().String(),
		inspection_task.ReaderFactoryGeneratorTaskID,
	},
	func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
		if taskMode == inspection_task.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}
		result, err := task.GetTypedVariableFromTaskVariable(v, form.AuditLogFilesForm.ID().String(), upload.UploadResult{})
		if err != nil {
			return nil, err
		}
		readerFactory, err := inspection_task.GetReaderFactoryFromTaskVariable(v)
		if err != nil {
			return "", err
		}
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
		metadataSet, err := common_task.GetMetadataSetFromVariable(v)
		if err != nil {
			return nil, err
		}
		header := metadataSet.LoadOrStore(header.HeaderMetadataKey, &header.HeaderMetadataFactory{}).(*header.Header)
		if len(logs) > 0 {
			header.StartTimeUnixSeconds = logs[0].Timestamp().Unix()
			header.EndTimeUnixSeconds = logs[len(logs)-1].Timestamp().Unix()
		}

		return logs, nil
	},
)

var OSSEventLogFilter = inspection_task.NewInspectionProcessor(
	OSSAuditLogEventLogsFilterTaskID,
	[]string{
		OSSAuditLogFileReaderTaskID,
	}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
		if taskMode == inspection_task.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}

		logs, err := task.GetTypedVariableFromTaskVariable(v, OSSAuditLogFileReaderTaskID, []*log.LogEntity{})
		if err != nil {
			return nil, err
		}

		var eventLogs []*log.LogEntity

		for _, l := range logs {
			if l.GetStringOrDefault("kind", "") == "Event" && l.GetStringOrDefault("responseObject.kind", "") == "Event" {
				l.LogType = enum.LogTypeEvent
				eventLogs = append(eventLogs, l)
			}
		}

		return eventLogs, nil
	})

var OSSNonEventLogFilter = inspection_task.NewInspectionProcessor(
	OSSAuditLogNonEventFilterTaskID,
	[]string{
		OSSAuditLogFileReaderTaskID,
	}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
		if taskMode == inspection_task.TaskModeDryRun {
			return []*log.LogEntity{}, nil
		}

		logs, err := task.GetTypedVariableFromTaskVariable(v, OSSAuditLogFileReaderTaskID, []*log.LogEntity{})
		if err != nil {
			return nil, err
		}

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
