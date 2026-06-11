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
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type taskProgressReporter struct {
	progress *inspectionmetadata.TaskProgressMetadata
}

func (t *taskProgressReporter) ReportProgress(percentage float32, status string) {
	t.progress.Update(percentage, status)
}

// SerializeTask is a subsequent task that must be included in the task graph after tasks like TimelineMapper and LogIngester.
// It retrieves the Builder instance populated by its preceding tasks and serializes its accumulated contents into the final KHI file.
var SerializeTask = inspectiontaskbase.NewProgressReportableInspectionTask(inspectioncore_contract.SerializerTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (*inspectioncore_contract.FileSystemStore, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		slog.DebugContext(ctx, "Skipping because this is in dryrun mode")
		return nil, nil
	}
	inspectionID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInspectionID)
	metadataSet := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	ioConfig := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentIOConfig)
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)

	// 1. Collect metadata to the v6 builder
	for _, key := range metadataSet.Keys() {
		metadata, found := typedmap.Get(metadataSet, inspectionmetadata.NewMetadataLabelsKey[inspectionmetadata.Metadata](key))
		if !found {
			return nil, fmt.Errorf("expected metadata not found: %s", key)
		}
		if err := builder.MetadataAccumulator.AddMetadata(metadata); err != nil {
			return nil, err
		}
	}

	// 2. Prepare Output File Store
	store := inspectioncore_contract.NewFileSystemInspectionResultRepository(filepath.Join(ioConfig.DataDestination, inspectionID+".khi"))
	writer, err := store.GetWriter()
	if err != nil {
		return nil, err
	}

	// 3. Build KHI v6 format and write
	if err := builder.Build(writer, &taskProgressReporter{progress: progress}); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// 4. Update final file size in metadata
	fileSize, err := store.GetInspectionResultSizeInBytes()
	if err != nil {
		return nil, err
	}

	header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey)
	if found {
		header.FileSize = fileSize
	}
	return store, nil
})
