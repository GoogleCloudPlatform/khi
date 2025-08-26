// Copyright 2024 Google LLC
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

package inspectioncore_impl

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var SerializeTask = inspectiontaskbase.NewProgressReportableInspectionTask(inspectioncore_contract.SerializerTaskID, []taskid.UntypedTaskReference{inspectioncore_contract.InspectionMainSubgraphDoneTaskID.Ref()}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (*inspectioncore_contract.FileSystemStore, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		slog.DebugContext(ctx, "Skipping because this is in dryrun mode")
		return nil, nil
	}
	inspectionID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInspectionID)
	metadataSet := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	ioConfig := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentIOConfig)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filepath.Join(ioConfig.DataDestination, inspectionID+".khi"))

	writer, err := store.GetWriter()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	resultMetadata, err := inspectionmetadata.GetSerializableSubsetMapFromMetadataSet(metadataSet, filter.NewEqualFilter(inspectionmetadata.LabelKeyIncludedInResultBinaryFlag, true, false))
	if err != nil {
		return nil, err
	}
	fileSize, err := builder.Finalize(ctx, resultMetadata, writer, progress)
	if err != nil {
		return nil, err
	}
	header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey)
	if found {
		header.FileSize = fileSize
	}
	return store, nil
})
