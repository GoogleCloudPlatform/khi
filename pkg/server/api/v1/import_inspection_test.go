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

package apiv1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/importinspection"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func createTestBinaryKHIBytes(t *testing.T, inspectionName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer, err := khifilev6.NewWriter(&buf)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	metadataChunk := &pb.MetadataChunk{
		Metadata: []*pb.MetadataItem{
			{
				Payload: &pb.MetadataItem_Header{
					Header: &pb.HeaderMetadata{
						InspectionName: proto.String(inspectionName),
						InspectionType: proto.String("gcp-gke"),
					},
				},
			},
		},
	}
	if err := writer.WriteChunk(khifilev6.ChunkTypeMetadata, metadataChunk); err != nil {
		t.Fatalf("failed to write metadata chunk: %v", err)
	}
	return buf.Bytes()
}

func setupTestServer(t *testing.T) (apiv1connect.ImportInspectionServiceClient, *coreinspection.InspectionTaskServer, func()) {
	t.Helper()
	tempDir := t.TempDir()
	destDir := t.TempDir()
	ioConfig := &inspectioncore_contract.IOConfig{
		TemporaryFolder: tempDir,
		DataDestination: destDir,
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	manager := importinspection.NewImportSessionManager(server, ioConfig)
	serviceServer := NewImportInspectionServiceServer(manager)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewImportInspectionServiceHandler(serviceServer)
	mux.Handle(path, handler)

	httpServer := httptest.NewServer(mux)
	client := apiv1connect.NewImportInspectionServiceClient(httpServer.Client(), httpServer.URL)

	cleanup := func() {
		httpServer.Close()
	}
	return client, server, cleanup
}

func TestImportInspectionService_FullLifecycle(t *testing.T) {
	testCases := []struct {
		name               string
		fileName           string
		inspectionName     string
		chunkSplitOffset   int
		wantInspectionName string
	}{
		{
			name:               "complete import lifecycle via Connect client",
			fileName:           "cluster.khi",
			inspectionName:     "Cluster-Production",
			chunkSplitOffset:   20,
			wantInspectionName: "Cluster-Production",
		},
		{
			name:               "complete import lifecycle with out-of-order chunk uploads",
			fileName:           "cluster-ooo.khi",
			inspectionName:     "Cluster-OOO",
			chunkSplitOffset:   15,
			wantInspectionName: "Cluster-OOO",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, inspectionServer, cleanup := setupTestServer(t)
			defer cleanup()

			ctx := context.Background()
			khiData := createTestBinaryKHIBytes(t, tc.inspectionName)

			// 1. StartImportInspection
			startResp, err := client.StartImportInspection(ctx, connect.NewRequest(&apiv1.StartImportInspectionRequest{
				FileName:       proto.String(tc.fileName),
				TotalSizeBytes: proto.Int64(int64(len(khiData))),
			}))
			if err != nil {
				t.Fatalf("StartImportInspection failed: %v", err)
			}
			token := startResp.Msg.GetImportToken()
			if token == "" {
				t.Fatal("StartImportInspection returned empty import token")
			}
			if startResp.Msg.GetSuggestedChunkSizeBytes() <= 0 {
				t.Errorf("SuggestedChunkSizeBytes should be positive, got %d", startResp.Msg.GetSuggestedChunkSizeBytes())
			}

			chunk1 := khiData[:tc.chunkSplitOffset]
			chunk2 := khiData[tc.chunkSplitOffset:]

			if tc.name == "complete import lifecycle with out-of-order chunk uploads" {
				// Upload Part 2 before Part 1
				uploadResp2, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(token),
					OffsetBytes: proto.Int64(int64(len(chunk1))),
					Data:        chunk2,
				}))
				if err != nil {
					t.Fatalf("UploadInspectionChunk part 2 failed: %v", err)
				}
				if uploadResp2.Msg.GetTotalReceivedBytes() != int64(len(chunk2)) {
					t.Fatalf("UploadInspectionChunk part 2 returned unexpected progress: received=%d", uploadResp2.Msg.GetTotalReceivedBytes())
				}

				uploadResp1, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(token),
					OffsetBytes: proto.Int64(0),
					Data:        chunk1,
				}))
				if err != nil {
					t.Fatalf("UploadInspectionChunk part 1 failed: %v", err)
				}
				if uploadResp1.Msg.GetTotalReceivedBytes() != int64(len(khiData)) {
					t.Fatalf("UploadInspectionChunk part 1 returned unexpected progress: received=%d", uploadResp1.Msg.GetTotalReceivedBytes())
				}
			} else {
				// 2. UploadInspectionChunk - Part 1
				uploadResp1, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(token),
					OffsetBytes: proto.Int64(0),
					Data:        chunk1,
				}))
				if err != nil {
					t.Fatalf("UploadInspectionChunk part 1 failed: %v", err)
				}
				if uploadResp1.Msg.GetTotalReceivedBytes() != int64(len(chunk1)) {
					t.Fatalf("UploadInspectionChunk part 1 returned unexpected progress: received=%d", uploadResp1.Msg.GetTotalReceivedBytes())
				}

				// 3. UploadInspectionChunk - Part 2
				uploadResp2, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(token),
					OffsetBytes: proto.Int64(int64(len(chunk1))),
					Data:        chunk2,
				}))
				if err != nil {
					t.Fatalf("UploadInspectionChunk part 2 failed: %v", err)
				}
				if uploadResp2.Msg.GetTotalReceivedBytes() != int64(len(khiData)) {
					t.Fatalf("UploadInspectionChunk part 2 returned unexpected progress: received=%d", uploadResp2.Msg.GetTotalReceivedBytes())
				}
			}

			// 4. CompleteImportInspection
			completeResp, err := client.CompleteImportInspection(ctx, connect.NewRequest(&apiv1.CompleteImportInspectionRequest{
				ImportToken: proto.String(token),
			}))
			if err != nil {
				t.Fatalf("CompleteImportInspection failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantInspectionName, completeResp.Msg.GetInspectionName()); diff != "" {
				t.Errorf("InspectionName mismatch (-want +got):\n%s", diff)
			}
			if completeResp.Msg.GetFileSizeBytes() != int64(len(khiData)) {
				t.Errorf("FileSizeBytes mismatch: got %d, want %d", completeResp.Msg.GetFileSizeBytes(), len(khiData))
			}

			// Verify that the inspection is available on the inspectionServer
			runner := inspectionServer.GetInspection(completeResp.Msg.GetInspectionId())
			if runner == nil {
				t.Fatalf("inspection not found on server: %s", completeResp.Msg.GetInspectionId())
			}
			if !runner.Started() {
				t.Errorf("expected runner.Started() to be true")
			}
		})
	}
}

func TestImportInspectionService_Errors(t *testing.T) {
	testCases := []struct {
		name     string
		execute  func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error
		wantCode connect.Code
	}{
		{
			name: "StartImportInspection with empty filename",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.StartImportInspection(ctx, connect.NewRequest(&apiv1.StartImportInspectionRequest{
					FileName: proto.String(""),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "StartImportInspection with non-positive total size",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.StartImportInspection(ctx, connect.NewRequest(&apiv1.StartImportInspectionRequest{
					FileName:       proto.String("test.khi"),
					TotalSizeBytes: proto.Int64(0),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "UploadInspectionChunk with empty token",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(""),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "UploadInspectionChunk with empty data",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				startResp, err := client.StartImportInspection(ctx, connect.NewRequest(&apiv1.StartImportInspectionRequest{
					FileName:       proto.String("test.khi"),
					TotalSizeBytes: proto.Int64(100),
				}))
				if err != nil {
					return err
				}
				_, err = client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String(startResp.Msg.GetImportToken()),
					OffsetBytes: proto.Int64(0),
					Data:        []byte{},
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "UploadInspectionChunk with non-existent token",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.UploadInspectionChunk(ctx, connect.NewRequest(&apiv1.UploadInspectionChunkRequest{
					ImportToken: proto.String("unknown-token"),
					OffsetBytes: proto.Int64(0),
					Data:        []byte("data"),
				}))
				return err
			},
			wantCode: connect.CodeNotFound,
		},
		{
			name: "CompleteImportInspection with non-existent token",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.CompleteImportInspection(ctx, connect.NewRequest(&apiv1.CompleteImportInspectionRequest{
					ImportToken: proto.String("unknown-token"),
				}))
				return err
			},
			wantCode: connect.CodeNotFound,
		},
		{
			name: "AbortImportInspection with non-existent token",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				_, err := client.AbortImportInspection(ctx, connect.NewRequest(&apiv1.AbortImportInspectionRequest{
					ImportToken: proto.String("unknown-token"),
				}))
				return err
			},
			wantCode: connect.CodeNotFound,
		},
		{
			name: "AbortImportInspection successfully aborts active session",
			execute: func(ctx context.Context, client apiv1connect.ImportInspectionServiceClient) error {
				startResp, err := client.StartImportInspection(ctx, connect.NewRequest(&apiv1.StartImportInspectionRequest{
					FileName:       proto.String("test.khi"),
					TotalSizeBytes: proto.Int64(100),
				}))
				if err != nil {
					return err
				}
				abortResp, err := client.AbortImportInspection(ctx, connect.NewRequest(&apiv1.AbortImportInspectionRequest{
					ImportToken: proto.String(startResp.Msg.GetImportToken()),
				}))
				if err != nil {
					return err
				}
				if !abortResp.Msg.GetAborted() {
					t.Errorf("expected aborted=true")
				}
				return nil
			},
			wantCode: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, cleanup := setupTestServer(t)
			defer cleanup()

			ctx := context.Background()
			err := tc.execute(ctx, client)

			if tc.wantCode == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error with code %v, got nil", tc.wantCode)
			}

			if connect.CodeOf(err) != tc.wantCode {
				t.Errorf("error code mismatch: got %v, want %v (error: %v)", connect.CodeOf(err), tc.wantCode, err)
			}
		})
	}
}
