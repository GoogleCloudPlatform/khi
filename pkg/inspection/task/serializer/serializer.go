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

package serializer

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/inspectiondata"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var SerializerTaskID = taskid.NewDefaultImplementationID[*inspectiondata.FileSystemStore](inspection_task.InspectionTaskPrefix + "serialize")

var SerializeTask = inspection_task.NewProgressReportableInspectionTask(SerializerTaskID, []taskid.UntypedTaskReference{inspection_task.InspectionMainSubgraphDoneTaskID.Ref(), ioconfig.IOConfigTaskID.Ref(), inspection_task.BuilderGeneratorTaskID.Ref()}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) (*inspectiondata.FileSystemStore, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		slog.DebugContext(ctx, "Skipping because this is in dryrun mode")
		return nil, nil
	}
	inspectionID := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInspectionID)
	metadataSet := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
	ioConfig := task.GetTaskResult(ctx, ioconfig.IOConfigTaskID.Ref())
	builder := task.GetTaskResult(ctx, inspection_task.BuilderGeneratorTaskID.Ref())
	store := inspectiondata.NewFileSystemInspectionResultRepository(filepath.Join(ioConfig.DataDestination, inspectionID+".khi"))

	writer, err := store.GetWriter()
	if err != nil {
		return nil, err
	}
	defer writer.Close()
	resultMetadata, err := metadata.GetSerializableSubsetMapFromMetadataSet(metadataSet, filter.NewEqualFilter(metadata.LabelKeyIncludedInResultBinaryFlag, true, false))
	if err != nil {
		return nil, err
	}
	fileSize, err := builder.Finalize(ctx, resultMetadata, writer, progress)
	if err != nil {
		return nil, err
	}
	header, found := typedmap.Get(metadataSet, header.HeaderMetadataKey)
	if found {
		header.FileSize = fileSize
	}
	return store, nil
})
