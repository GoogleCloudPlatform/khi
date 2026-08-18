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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func createTestKHIFile(t *testing.T, dir string, metadataItems []*pb.MetadataItem) string {
	t.Helper()
	filePath := filepath.Join(dir, "test.khi")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()

	writer, err := khifilev6.NewWriter(file)
	if err != nil {
		t.Fatalf("failed to create KHI writer: %v", err)
	}

	if metadataItems != nil {
		metadataChunk := &pb.MetadataChunk{
			Metadata: metadataItems,
		}
		if err := writer.WriteChunk(khifilev6.ChunkTypeMetadata, metadataChunk); err != nil {
			t.Fatalf("failed to write metadata chunk: %v", err)
		}
	}

	return filePath
}

func TestValidateAndExtractMetadata(t *testing.T) {
	testCases := []struct {
		name               string
		setupFile          func(t *testing.T, dir string) string
		wantInspectionName string
		wantInspectionType string
		wantQueryCount     int
		wantErr            bool
	}{
		{
			name: "valid KHI file with full metadata",
			setupFile: func(t *testing.T, dir string) string {
				items := []*pb.MetadataItem{
					{
						Payload: &pb.MetadataItem_Header{
							Header: &pb.HeaderMetadata{
								InspectionName: proto.String("Cluster-Alpha"),
								InspectionType: proto.String("gcp-gke"),
							},
						},
					},
					{
						Payload: &pb.MetadataItem_Query{
							Query: &pb.QueryMetadata{
								Queries: []*pb.QueryItem{
									{
										Id:    proto.String("q1"),
										Name:  proto.String("Audit Logs"),
										Query: proto.String("resource.type=\"k8s_cluster\""),
									},
								},
							},
						},
					},
				}
				return createTestKHIFile(t, dir, items)
			},
			wantInspectionName: "Cluster-Alpha",
			wantInspectionType: "gcp-gke",
			wantQueryCount:     1,
			wantErr:            false,
		},
		{
			name: "KHI file without header metadata",
			setupFile: func(t *testing.T, dir string) string {
				return createTestKHIFile(t, dir, nil)
			},
			wantErr: true,
		},
		{
			name: "corrupted non-KHI file",
			setupFile: func(t *testing.T, dir string) string {
				filePath := filepath.Join(dir, "corrupted.khi")
				if err := os.WriteFile(filePath, []byte("NOT_A_KHI_FILE_DATA"), 0644); err != nil {
					t.Fatalf("failed to write dummy file: %v", err)
				}
				return filePath
			},
			wantErr: true,
		},
		{
			name: "non-existent file",
			setupFile: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "non_existent.khi")
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetPath := tc.setupFile(t, tmpDir)

			header, metadataMap, err := ValidateAndExtractMetadata(targetPath)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateAndExtractMetadata() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.wantInspectionName, header.InspectionName); diff != "" {
				t.Errorf("InspectionName mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantInspectionType, header.InspectionType); diff != "" {
				t.Errorf("InspectionType mismatch (-want +got):\n%s", diff)
			}

			queryMetadata, found := typedmap.Get(metadataMap, inspectionmetadata.QueryMetadataKey)
			if tc.wantQueryCount > 0 {
				if !found {
					t.Errorf("expected QueryMetadata to be found")
				} else if len(queryMetadata.Queries) != tc.wantQueryCount {
					t.Errorf("query count mismatch: got %d, want %d", len(queryMetadata.Queries), tc.wantQueryCount)
				}
			}

			progressMetadata, found := typedmap.Get(metadataMap, inspectionmetadata.ProgressMetadataKey)
			if !found {
				t.Errorf("expected ProgressMetadata to be present")
			} else if progressMetadata.Phase != inspectionmetadata.TaskPhaseDone {
				t.Errorf("progress phase mismatch: got %v, want %v", progressMetadata.Phase, inspectionmetadata.TaskPhaseDone)
			}
		})
	}
}

func TestMetadataChunks(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(t *testing.T) *khifilev6.Reader
		wantChunks int
		wantErr    bool
	}{
		{
			name: "single metadata chunk",
			setup: func(t *testing.T) *khifilev6.Reader {
				var buf bytes.Buffer
				writer, err := khifilev6.NewWriter(&buf)
				if err != nil {
					t.Fatalf("failed to create writer: %v", err)
				}
				metadata := &pb.MetadataChunk{
					Metadata: []*pb.MetadataItem{
						{
							Payload: &pb.MetadataItem_Header{
								Header: &pb.HeaderMetadata{
									InspectionName: proto.String("test"),
								},
							},
						},
					},
				}
				if err := writer.WriteChunk(khifilev6.ChunkTypeMetadata, metadata); err != nil {
					t.Fatalf("failed to write metadata chunk: %v", err)
				}
				r, err := khifilev6.NewReader(&buf)
				if err != nil {
					t.Fatalf("failed to create reader: %v", err)
				}
				return r
			},
			wantChunks: 1,
			wantErr:    false,
		},
		{
			name: "metadata chunk and non-metadata chunk interleaved",
			setup: func(t *testing.T) *khifilev6.Reader {
				var buf bytes.Buffer
				writer, err := khifilev6.NewWriter(&buf)
				if err != nil {
					t.Fatalf("failed to create writer: %v", err)
				}
				logMsg := &pb.Log{
					Id: proto.Uint32(1),
				}
				if err := writer.WriteChunk(khifilev6.ChunkTypeLog, logMsg); err != nil {
					t.Fatalf("failed to write log chunk: %v", err)
				}
				metadata := &pb.MetadataChunk{
					Metadata: []*pb.MetadataItem{
						{
							Payload: &pb.MetadataItem_Header{
								Header: &pb.HeaderMetadata{
									InspectionName: proto.String("test-2"),
								},
							},
						},
					},
				}
				if err := writer.WriteChunk(khifilev6.ChunkTypeMetadata, metadata); err != nil {
					t.Fatalf("failed to write metadata chunk: %v", err)
				}
				r, err := khifilev6.NewReader(&buf)
				if err != nil {
					t.Fatalf("failed to create reader: %v", err)
				}
				return r
			},
			wantChunks: 1,
			wantErr:    false,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) *khifilev6.Reader {
				var buf bytes.Buffer
				_, err := khifilev6.NewWriter(&buf)
				if err != nil {
					t.Fatalf("failed to create writer: %v", err)
				}
				r, err := khifilev6.NewReader(&buf)
				if err != nil {
					t.Fatalf("failed to create reader: %v", err)
				}
				return r
			},
			wantChunks: 0,
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.setup(t)
			var gotChunks []*pb.MetadataChunk
			var gotErr error
			for chunk, err := range metadataChunks(r) {
				if err != nil {
					gotErr = err
					break
				}
				gotChunks = append(gotChunks, chunk)
			}

			if (gotErr != nil) != tc.wantErr {
				t.Fatalf("metadataChunks() error = %v, wantErr = %v", gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantChunks, len(gotChunks)); diff != "" {
				t.Errorf("chunk count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
