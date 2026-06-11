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

package inspectioncore_impl

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/proto"
)

// TestSerializeTask verifies that SerializeTask successfully converts accumulated
// builder contents into the v6 protobuf chunk format.
func TestSerializeTask(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(ctx context.Context, b *khifilev6.Builder)
		verify   func(t *testing.T, store *inspectioncore_contract.FileSystemStore)
		wantErr  bool
		taskMode inspectioncore_contract.InspectionTaskModeType
	}{
		{
			name: "successfully serializes v6 format containing metadata chunks",
			setup: func(ctx context.Context, b *khifilev6.Builder) {
				// Add a dummy header metadata.
				_ = b.MetadataAccumulator.AddMetadata(&inspectionmetadata.HeaderMetadata{
					InspectionType: "test-type",
					InspectionName: "test-name",
				})
			},
			verify: func(t *testing.T, store *inspectioncore_contract.FileSystemStore) {
				if store == nil {
					t.Fatal("Store should not be nil.")
				}

				reader, err := store.GetReader()
				if err != nil {
					t.Fatalf("Failed to get reader: %v.", err)
				}
				defer reader.Close()

				khiReader, err := khifilev6.NewReader(reader)
				if err != nil {
					t.Fatalf("Failed to initialize KHI reader: %v.", err)
				}

				foundMetadata := false
				foundTestHeader := false

				for {
					chunk, err := khiReader.NextChunk()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						t.Fatalf("Failed to read next chunk: %v.", err)
					}

					if chunk.Type == khifilev6.ChunkTypeMetadata {
						foundMetadata = true
						var metadataChunk pb.MetadataChunk
						if err := proto.Unmarshal(chunk.Data, &metadataChunk); err != nil {
							t.Fatalf("Failed to unmarshal metadata chunk: %v.", err)
						}
						for _, item := range metadataChunk.Metadata {
							if header := item.GetHeader(); header != nil {
								if header.GetInspectionType() == "test-type" && header.GetInspectionName() == "test-name" {
									foundTestHeader = true
									break
								}
							}
						}
					}
				}

				if !foundMetadata {
					t.Error("Expected MetadataChunk, but it was not found in the output file.")
				}
				if !foundTestHeader {
					t.Error("Could not find the expected header metadata with type 'test-type' and name 'test-name' inside the MetadataChunk.")
				}
			},
			wantErr:  false,
			taskMode: inspectioncore_contract.TaskModeRun,
		},
		{
			name: "skips serialization and returns nil store in DryRun mode",
			setup: func(ctx context.Context, b *khifilev6.Builder) {
				_ = b.MetadataAccumulator.AddMetadata(&inspectionmetadata.HeaderMetadata{
					InspectionType: "test-type",
					InspectionName: "test-name",
				})
			},
			verify: func(t *testing.T, store *inspectioncore_contract.FileSystemStore) {
				if store != nil {
					t.Errorf("Expected nil store in DryRun mode, got %v.", store)
				}
			},
			wantErr:  false,
			taskMode: inspectioncore_contract.TaskModeDryRun,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			taskCtx := inspectiontest.WithDefaultTestInspectionTaskContext(ctx)

			// Obtain the builder from task context.
			builder := khictx.MustGetValue(taskCtx, inspectioncore_contract.Builder)

			if tc.setup != nil {
				tc.setup(taskCtx, builder)
			}

			mode := tc.taskMode
			store, _, err := inspectiontest.RunInspectionTask(taskCtx, SerializeTask, mode, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SerializeTask error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				tc.verify(t, store)
				// Cleanup the generated file.
				if store != nil {
					reader, err := store.GetReader()
					if err == nil {
						if file, ok := reader.(*os.File); ok {
							path := file.Name()
							_ = file.Close()
							_ = os.Remove(path)
						} else {
							_ = reader.Close()
						}
					}
				}
			}
		})
	}
}
