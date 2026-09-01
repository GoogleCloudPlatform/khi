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
	"io"
	"slices"
	"strings"
	"unsafe"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

var (
	pathAuditFileReaderStage          = structured.CompileFieldPath("stage")
	pathAuditFileReaderStageTimestamp = structured.CompileFieldPath("stageTimestamp")
)

var AuditLogFileReaderTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	ossclusterk8s_contract.AuditLogFileReaderTaskID,
	[]taskid.UntypedTaskReference{
		ossclusterk8s_contract.InputAuditLogFilesFormTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return []*log.Log{}, nil
		}
		result := coretask.GetTaskResult(ctx, ossclusterk8s_contract.InputAuditLogFilesFormTaskID.Ref())

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

		progressutil.ReportProgressFromArraySync(tp, logLines, func(i int, line string) error {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				return nil
			}

			node := structured.NewLazyJSONNodeFromBytes(unsafe.Slice(unsafe.StringData(trimmed), len(trimmed)))
			reader := structured.NewNodeReader(node)

			// TODO: we may need to consider processing logs not with ResponseComplete stage. All logs not on the ResponseComplete stage will be ignored for now.
			if reader.ReadStringOrDefault(pathAuditFileReaderStage, "") != "ResponseComplete" {
				return nil
			}

			ts, _ := reader.ReadTimestamp(pathAuditFileReaderStageTimestamp)
			l := log.NewLogWithTimestamp(reader, ts)
			logs = append(logs, l)
			return nil
		})

		slices.SortFunc(logs, func(a, b *log.Log) int {
			return a.Timestamp.Compare(b.Timestamp)
		})
		metadataSet := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		header := typedmap.GetOrDefault(metadataSet, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{})

		if len(logs) > 0 {
			header.StartTimeUnixSeconds = logs[0].Timestamp.Unix()
			header.EndTimeUnixSeconds = logs[len(logs)-1].Timestamp.Unix()
		}

		return logs, nil
	},
)
