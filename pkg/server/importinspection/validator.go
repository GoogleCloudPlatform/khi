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

package importinspection

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"google.golang.org/protobuf/proto"
)

// ValidateAndExtractMetadata validates that the file at the given path is a valid KHI v6 binary file
// and extracts the header metadata and general metadata map needed by InspectionTaskServer.
func ValidateAndExtractMetadata(filePath string) (*inspectionmetadata.HeaderMetadata, *typedmap.ReadonlyTypedMap, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	khiReader, err := khifilev6.NewReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid KHI file container: %w", err)
	}

	metadataMap := typedmap.NewTypedMap()
	var extractedHeader *inspectionmetadata.HeaderMetadata

	for metadataChunk, err := range metadataChunks(khiReader) {
		if err != nil {
			return nil, nil, err
		}

		for _, item := range metadataChunk.Metadata {
			if pbHeader := item.GetHeader(); pbHeader != nil {
				extractedHeader = &inspectionmetadata.HeaderMetadata{
					InspectionType:         pbHeader.GetInspectionType(),
					InspectionName:         pbHeader.GetInspectionName(),
					InspectionTypeIconPath: pbHeader.GetInspectionTypeIconPath(),
					StartTimeUnixSeconds:   pbHeader.GetStartTimeUnixSeconds(),
					EndTimeUnixSeconds:     pbHeader.GetEndTimeUnixSeconds(),
					InspectTimeUnixSeconds: pbHeader.GetInspectTimeUnixSeconds(),
					SuggestedFileName:      pbHeader.GetSuggestedFilename(),
					FileSize:               int(pbHeader.GetFileSize()),
				}
				typedmap.Set(metadataMap, inspectionmetadata.HeaderMetadataKey, extractedHeader)
			}
			if pbQuery := item.GetQuery(); pbQuery != nil {
				queryMetadata := inspectionmetadata.NewQueryMetadata()
				for _, q := range pbQuery.GetQueries() {
					queryMetadata.SetQuery(q.GetId(), q.GetName(), q.GetQuery())
				}
				typedmap.Set(metadataMap, inspectionmetadata.QueryMetadataKey, queryMetadata)
			}
		}
	}

	if extractedHeader == nil {
		return nil, nil, fmt.Errorf("header metadata not found in KHI file")
	}

	// Initialize default progress and error metadata for imported inspections.
	progress := inspectionmetadata.NewProgress()
	progress.MarkDone()
	typedmap.Set(metadataMap, inspectionmetadata.ProgressMetadataKey, progress)

	errorMetadata := inspectionmetadata.NewErrorMessageSetMetadata()
	typedmap.Set(metadataMap, inspectionmetadata.ErrorMessageSetMetadataKey, errorMetadata)

	return extractedHeader, metadataMap.AsReadonly(), nil
}

// metadataChunks iterates over all MetadataChunk messages from the KHI reader.
func metadataChunks(khiReader *khifilev6.Reader) iter.Seq2[*pb.MetadataChunk, error] {
	return func(yield func(*pb.MetadataChunk, error) bool) {
		for {
			chunk, err := khiReader.NextChunk()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(nil, fmt.Errorf("failed to read chunk from KHI file: %w", err))
				return
			}

			if chunk.Type == khifilev6.ChunkTypeMetadata {
				var metadataChunk pb.MetadataChunk
				if err := proto.Unmarshal(chunk.Data, &metadataChunk); err != nil {
					yield(nil, fmt.Errorf("failed to unmarshal metadata chunk: %w", err))
					return
				}
				if !yield(&metadataChunk, nil) {
					return
				}
			}
		}
	}
}
