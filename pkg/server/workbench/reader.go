// Copyright 2026 Google LLC
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

package workbench

import (
	"context"
	"errors"
	"fmt"
	"io"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"google.golang.org/protobuf/proto"
)

// ProgressCallback receives streaming progress updates during dataset loading.
type ProgressCallback func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error

// NewWorkbenchFromReader creates and initializes a Workbench instance by parsing chunks from the given reader.
func NewWorkbenchFromReader(
	ctx context.Context,
	id string,
	inspectionID string,
	reader io.Reader,
	totalSize int64,
	onProgress ProgressCallback,
) (*Workbench, error) {
	khiReader, err := khifilev6model.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create KHI file reader: %w", err)
	}

	wb := NewWorkbench(id, inspectionID)
	var processedBytes int64

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		chunk, err := khiReader.NextChunk()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read next chunk: %w", err)
		}

		if totalSize > 0 {
			processedBytes += int64(len(chunk.Data))
			pct := 10.0 + (float64(processedBytes)/float64(totalSize))*70.0
			if pct > 80.0 {
				pct = 80.0
			}
			if onProgress != nil {
				if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_PARSING_CHUNKS, pct, fmt.Sprintf("Parsing chunks (type %d)...", chunk.Type)); err != nil {
					return nil, err
				}
			}
		}

		if err := wb.ingestChunk(chunk); err != nil {
			return nil, fmt.Errorf("failed to ingest chunk: %w", err)
		}
	}

	if onProgress != nil {
		if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 90, "Finalizing in-memory dataset index..."); err != nil {
			return nil, err
		}
	}

	searchIndex, err := wb.BuildSearchIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to build search index: %w", err)
	}
	wb.searchIndex = searchIndex

	return wb, nil
}

func (w *Workbench) ingestChunk(chunk *khifilev6model.Chunk) error {
	switch chunk.Type {
	case khifilev6model.ChunkTypeMetadata:
		var meta khifilev6.MetadataChunk
		if err := proto.Unmarshal(chunk.Data, &meta); err != nil {
			return err
		}
		w.metadataChunks = append(w.metadataChunks, &meta)
	case khifilev6model.ChunkTypeInternPool:
		var pool khifilev6.InterningPoolChunk
		if err := proto.Unmarshal(chunk.Data, &pool); err != nil {
			return err
		}
		w.internPool.IngestChunk(&pool)
	case khifilev6model.ChunkTypeTimelineStyle:
		var style khifilev6.TimelineStyleChunk
		if err := proto.Unmarshal(chunk.Data, &style); err != nil {
			return err
		}
		w.styleChunk = &style
	case khifilev6model.ChunkTypeLog:
		var logChunk khifilev6.LogChunk
		if err := proto.Unmarshal(chunk.Data, &logChunk); err != nil {
			return err
		}
		w.logChunks = append(w.logChunks, &logChunk)
	case khifilev6model.ChunkTypeTimeline:
		var timelineChunk khifilev6.TimelineChunk
		if err := proto.Unmarshal(chunk.Data, &timelineChunk); err != nil {
			return err
		}
		w.timelineChunks = append(w.timelineChunks, &timelineChunk)
	}
	return nil
}
