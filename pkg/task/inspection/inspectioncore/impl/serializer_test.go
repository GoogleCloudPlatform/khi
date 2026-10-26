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
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// TestSerializeTask verifies that serializerTask successfully converts accumulated
// builder contents into the v6 protobuf chunk format.
func TestSerializeTask(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(ctx context.Context, b *khifilev6.Builder)
		verify   func(t *testing.T, metadata *typedmap.ReadonlyTypedMap, store *inspectioncore.FileSystemStore)
		wantErr  bool
		taskMode inspectioncore.InspectionTaskModeType
	}{
		{
			name: "successfully serializes v6 format containing metadata chunks and filter vocabulary",
			setup: func(ctx context.Context, b *khifilev6.Builder) {
				// Add a dummy header metadata.
				_ = b.MetadataAccumulator.AddMetadata(&inspectionmetadata.HeaderMetadata{
					InspectionType: "test-type",
					InspectionName: "test-name",
				})

				kindType := &pb.TimelineType{
					Id:           proto.Uint32(1),
					Label:        proto.String("kind"),
					SortPriority: proto.Int32(200),
				}
				_ = b.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
					Name: "pod",
					Type: kindType,
				})

				idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
				node := structured.NewStandardMap(
					[]string{"msg"},
					[]structured.Node{structured.NewStandardScalarNode("test")},
				)
				_ = b.LogAccumulator.AddLog(&khifilev6.StagingLog{
					Log:      log.NewLog(idGen, structured.NewNodeReader(node)),
					Summary:  "summary",
					Severity: &pb.Severity{Id: proto.Uint32(1)},
					LogType:  &pb.LogType{Id: proto.Uint32(1), Label: proto.String("k8s_audit")},
				})
			},
			verify: func(t *testing.T, metadata *typedmap.ReadonlyTypedMap, store *inspectioncore.FileSystemStore) {
				if store == nil {
					t.Fatal("Store should not be nil.")
				}

				wantVocabulary := inspectionmetadata.FilterVocabulary{
					TimelineTypes: []inspectionmetadata.TimelineTypeVocabulary{
						{Label: "kind", NameCount: 1, ExampleNames: []string{"pod"}},
					},
					LogTypes: []string{"k8s_audit"},
				}
				vocabMeta, found := typedmap.Get(metadata, inspectionmetadata.FilterVocabularyMetadataKey)
				if !found {
					t.Fatal("Expected FilterVocabularyMetadata in metadata map.")
				}
				if diff := cmp.Diff(wantVocabulary, vocabMeta.Get()); diff != "" {
					t.Errorf("FilterVocabularyMetadata mismatch (-want +got):\n%s", diff)
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
				var gotVocabProto *pb.FilterVocabularyMetadata

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
								}
							}
							if vocab := item.GetFilterVocabulary(); vocab != nil {
								gotVocabProto = vocab
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
				wantVocabProto := &pb.FilterVocabularyMetadata{
					TimelineTypes: []*pb.TimelineTypeVocabulary{
						{
							Label:        proto.String("kind"),
							NameCount:    proto.Uint32(1),
							ExampleNames: []string{"pod"},
						},
					},
					LogTypes: []string{"k8s_audit"},
				}
				if diff := cmp.Diff(wantVocabProto, gotVocabProto, protocmp.Transform()); diff != "" {
					t.Errorf("FilterVocabularyMetadata proto mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr:  false,
			taskMode: inspectioncore.TaskModeRun,
		},
		{
			name: "skips serialization and returns nil store in DryRun mode",
			setup: func(ctx context.Context, b *khifilev6.Builder) {
				_ = b.MetadataAccumulator.AddMetadata(&inspectionmetadata.HeaderMetadata{
					InspectionType: "test-type",
					InspectionName: "test-name",
				})
			},
			verify: func(t *testing.T, metadata *typedmap.ReadonlyTypedMap, store *inspectioncore.FileSystemStore) {
				if store != nil {
					t.Errorf("Expected nil store in DryRun mode, got %v.", store)
				}
			},
			wantErr:  false,
			taskMode: inspectioncore.TaskModeDryRun,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			taskCtx := inspectiontest.WithDefaultTestInspectionTaskContext(ctx)

			if tc.taskMode == inspectioncore.TaskModeRun {
				ioConfig := khictx.MustGetValue(taskCtx, inspectioncore.CurrentIOConfig)
				inspectionID := khictx.MustGetValue(taskCtx, inspectioncore.InspectionTaskInspectionID)
				store := inspectioncore.NewFileSystemInspectionResultRepository(filepath.Join(ioConfig.DataDestination, inspectionID+".khi"))
				fw, err := store.GetWriter()
				if err != nil {
					t.Fatalf("failed to create file writer: %v", err)
				}
				w, err := khifilev6.NewWriter(fw)
				if err != nil {
					_ = fw.Close()
					t.Fatalf("failed to create writer: %v", err)
				}
				idGen := khictx.MustGetValue(taskCtx, inspectioncore.IDGenerator)
				b := khifilev6.NewBuilder(idGen, w)
				taskCtx = khictx.WithValue(taskCtx, inspectioncore.Builder, b)
			}

			// Obtain the builder from task context.
			builder := khictx.MustGetValue(taskCtx, inspectioncore.Builder)

			if tc.setup != nil {
				tc.setup(taskCtx, builder)
			}

			mode := tc.taskMode
			store, metadata, err := inspectiontest.Run(t, taskCtx, serializerTask, mode, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SerializeTask error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				tc.verify(t, metadata, store)
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
