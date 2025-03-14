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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/inspectiondata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const SerializerTaskID = inspection_task.InspectionTaskPrefix + "serialize"

var SerializeTask = inspection_task.NewInspectionProcessor(SerializerTaskID, []string{inspection_task.InspectionMainSubgraphName + "-done", ioconfig.IOConfigTaskName, inspection_task.BuilderGeneratorTaskID}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
	if taskMode == inspection_task.TaskModeDryRun {
		slog.DebugContext(ctx, "Skipping because this is in dryrun mode")
		return nil, nil
	}
	taskId, err := inspection_task.GetInspectionIdFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	ioConfig, err := ioconfig.GetIOConfigFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	store := inspectiondata.NewFileSystemInspectionResultRepository(filepath.Join(ioConfig.DataDestination, taskId+".khi"))
	writer, err := store.GetWriter()
	if err != nil {
		return nil, err
	}
	metadataSet, err := inspection_task.GetMetadataSetFromVariable(v)
	if err != nil {
		return nil, err
	}
	resultMetadata, err := metadata.GetSerializableSubsetMapFromMetadataSet(metadataSet, filter.NewEqualFilter(metadata.LabelKeyIncludedInResultBinaryFlag, true, false))
	if err != nil {
		return nil, err
	}
	fileSize, err := builder.Finalize(ctx, resultMetadata, writer, progress)
	if err != nil {
		return nil, err
	}
	err = store.Close()
	if err != nil {
		return nil, err
	}
	header, found := typedmap.Get(metadataSet, header.HeaderMetadataKey)
	if found {
		header.FileSize = fileSize
	}
	return store, nil
})
