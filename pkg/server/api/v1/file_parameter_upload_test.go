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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func setupTestFileUploadServer(t *testing.T) (apiv1connect.FileParameterUploadServiceClient, *upload.UploadFileStore, *upload.LocalUploadFileStoreProvider, func()) {
	t.Helper()
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "store")
	uploadDir := filepath.Join(tempDir, "upload")

	provider := upload.NewLocalUploadFileStoreProvider(storeDir)
	uploadStore := upload.NewUploadFileStore(provider)
	chunkManager := chunkedupload.NewChunkSessionManager(uploadDir)

	manager := upload.NewFileParameterUploadManager(uploadStore, chunkManager)
	serviceServer := NewFileParameterUploadServiceServer(manager)

	mux := http.NewServeMux()
	path, handler := apiv1connect.NewFileParameterUploadServiceHandler(serviceServer)
	mux.Handle(path, handler)

	httpServer := httptest.NewServer(mux)
	client := apiv1connect.NewFileParameterUploadServiceClient(httpServer.Client(), httpServer.URL)

	cleanup := func() {
		httpServer.Close()
		chunkManager.Close()
	}
	return client, uploadStore, provider, cleanup
}

func TestFileParameterUploadService_Lifecycle(t *testing.T) {
	testCases := []struct {
		name         string
		fileName     string
		fileData     []byte
		chunkSplitAt int
	}{
		{
			name:         "upload single file in 2 chunks successfully",
			fileName:     "audit.log",
			fileData:     []byte("hello world 12345"),
			chunkSplitAt: 8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, uploadStore, provider, cleanup := setupTestFileUploadServer(t)
			defer cleanup()

			ctx := context.Background()
			token := uploadStore.GetUploadToken("field-upload-test", nil, "field-param")

			// 1. Start file upload
			startResp, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
				UploadTokenId:  proto.String(token.GetID()),
				FileName:       proto.String(tc.fileName),
				TotalSizeBytes: proto.Int64(int64(len(tc.fileData))),
			}))
			if err != nil {
				t.Fatalf("StartFileUpload failed: %v", err)
			}
			sessionToken := startResp.Msg.GetSessionToken()
			if sessionToken == "" {
				t.Fatalf("expected non-empty session token")
			}

			// 2. Upload chunk 1
			chunk1 := tc.fileData[:tc.chunkSplitAt]
			chunk1Resp, err := client.UploadFileChunk(ctx, connect.NewRequest(&apiv1.UploadFileChunkRequest{
				SessionToken: proto.String(sessionToken),
				OffsetBytes:  proto.Int64(0),
				Data:         chunk1,
			}))
			if err != nil {
				t.Fatalf("UploadFileChunk 1 failed: %v", err)
			}
			if chunk1Resp.Msg.GetTotalReceivedBytes() != int64(len(chunk1)) {
				t.Errorf("chunk 1 received mismatch (-want +got):\n%s",
					cmp.Diff(int64(len(chunk1)), chunk1Resp.Msg.GetTotalReceivedBytes()))
			}

			// 3. Upload chunk 2
			chunk2 := tc.fileData[tc.chunkSplitAt:]
			chunk2Resp, err := client.UploadFileChunk(ctx, connect.NewRequest(&apiv1.UploadFileChunkRequest{
				SessionToken: proto.String(sessionToken),
				OffsetBytes:  proto.Int64(int64(tc.chunkSplitAt)),
				Data:         chunk2,
			}))
			if err != nil {
				t.Fatalf("UploadFileChunk 2 failed: %v", err)
			}
			if chunk2Resp.Msg.GetTotalReceivedBytes() != int64(len(tc.fileData)) {
				t.Errorf("chunk 2 received mismatch (-want +got):\n%s",
					cmp.Diff(int64(len(tc.fileData)), chunk2Resp.Msg.GetTotalReceivedBytes()))
			}

			// 4. Complete upload
			completeResp, err := client.CompleteFileUpload(ctx, connect.NewRequest(&apiv1.CompleteFileUploadRequest{
				SessionToken: proto.String(sessionToken),
			}))
			if err != nil {
				t.Fatalf("CompleteFileUpload failed: %v", err)
			}
			if completeResp.Msg.GetFileSizeBytes() != int64(len(tc.fileData)) {
				t.Errorf("file size mismatch (-want +got):\n%s",
					cmp.Diff(int64(len(tc.fileData)), completeResp.Msg.GetFileSizeBytes()))
			}

			// 5. Verify persisted file content in provider
			reader, err := provider.Read(token)
			if err != nil {
				t.Fatalf("provider.Read failed: %v", err)
			}
			defer reader.Close()

			content, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("io.ReadAll failed: %v", err)
			}
			if diff := cmp.Diff(string(tc.fileData), string(content)); diff != "" {
				t.Errorf("content mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFileParameterUploadService_Abort(t *testing.T) {
	client, uploadStore, _, cleanup := setupTestFileUploadServer(t)
	defer cleanup()

	ctx := context.Background()
	token := uploadStore.GetUploadToken("field-upload-abort", nil, "field-param")

	startResp, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
		UploadTokenId:  proto.String(token.GetID()),
		FileName:       proto.String("sample.log"),
		TotalSizeBytes: proto.Int64(100),
	}))
	if err != nil {
		t.Fatalf("StartFileUpload failed: %v", err)
	}
	sessionToken := startResp.Msg.GetSessionToken()

	abortResp, err := client.AbortFileUpload(ctx, connect.NewRequest(&apiv1.AbortFileUploadRequest{
		SessionToken: proto.String(sessionToken),
	}))
	if err != nil {
		t.Fatalf("AbortFileUpload failed: %v", err)
	}
	if !abortResp.Msg.GetAborted() {
		t.Errorf("expected aborted=true")
	}

	// Second abort should return NotFound
	_, err = client.AbortFileUpload(ctx, connect.NewRequest(&apiv1.AbortFileUploadRequest{
		SessionToken: proto.String(sessionToken),
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", err)
	}
}

func TestFileParameterUploadService_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name     string
		run      func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error
		wantCode connect.Code
	}{
		{
			name: "start with missing upload token ID",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
					UploadTokenId:  proto.String(""),
					FileName:       proto.String("test.log"),
					TotalSizeBytes: proto.Int64(10),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "start with empty file name",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
					UploadTokenId:  proto.String("tok-1"),
					FileName:       proto.String(""),
					TotalSizeBytes: proto.Int64(10),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "start with non-positive total size",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
					UploadTokenId:  proto.String("tok-1"),
					FileName:       proto.String("test.log"),
					TotalSizeBytes: proto.Int64(0),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "start with non-existent upload token ID",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.StartFileUpload(ctx, connect.NewRequest(&apiv1.StartFileUploadRequest{
					UploadTokenId:  proto.String("non-existent-token"),
					FileName:       proto.String("test.log"),
					TotalSizeBytes: proto.Int64(10),
				}))
				return err
			},
			wantCode: connect.CodeNotFound,
		},
		{
			name: "upload chunk with empty session token",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.UploadFileChunk(ctx, connect.NewRequest(&apiv1.UploadFileChunkRequest{
					SessionToken: proto.String(""),
					OffsetBytes:  proto.Int64(0),
					Data:         []byte("data"),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "complete with empty session token",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.CompleteFileUpload(ctx, connect.NewRequest(&apiv1.CompleteFileUploadRequest{
					SessionToken: proto.String(""),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "abort with empty session token",
			run: func(ctx context.Context, client apiv1connect.FileParameterUploadServiceClient) error {
				_, err := client.AbortFileUpload(ctx, connect.NewRequest(&apiv1.AbortFileUploadRequest{
					SessionToken: proto.String(""),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, _, cleanup := setupTestFileUploadServer(t)
			defer cleanup()

			ctx := context.Background()
			err := tc.run(ctx, client)
			if gotCode := connect.CodeOf(err); gotCode != tc.wantCode {
				t.Errorf("error code mismatch: want %v, got %v (err: %v)", tc.wantCode, gotCode, err)
			}
		})
	}
}
