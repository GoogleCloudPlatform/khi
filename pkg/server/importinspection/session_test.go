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
	"errors"
	"os"
	"sync"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
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

func TestImportSessionManager_Lifecycle(t *testing.T) {
	testCases := []struct {
		name               string
		chunkSize          int
		inspectionName     string
		wantInspectionName string
	}{
		{
			name:               "successfully uploads file sequentially in 2 chunks and completes registration",
			chunkSize:          16,
			inspectionName:     "Inspection-123",
			wantInspectionName: "Inspection-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			manager := NewImportSessionManager(server, ioConfig)
			defer manager.Close()

			data := createTestBinaryKHIBytes(t, tc.inspectionName)

			session, err := manager.StartSession("test.khi", int64(len(data)))
			if err != nil {
				t.Fatalf("StartSession failed: %v", err)
			}

			// Split data into chunks
			chunk1 := data[:tc.chunkSize]
			chunk2 := data[tc.chunkSize:]

			n1, err := manager.WriteChunk(session.Token, 0, chunk1)
			if err != nil {
				t.Fatalf("WriteChunk 0 failed: %v", err)
			}
			if n1 != int64(len(chunk1)) {
				t.Fatalf("WriteChunk 0 unexpected return: n=%d", n1)
			}

			n2, err := manager.WriteChunk(session.Token, int64(len(chunk1)), chunk2)
			if err != nil {
				t.Fatalf("WriteChunk 1 failed: %v", err)
			}
			if n2 != int64(len(data)) {
				t.Fatalf("WriteChunk 1 unexpected return: n=%d", n2)
			}

			result, err := manager.CompleteSession(session.Token)
			if err != nil {
				t.Fatalf("CompleteSession failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantInspectionName, result.InspectionName); diff != "" {
				t.Errorf("InspectionName mismatch (-want +got):\n%s", diff)
			}

			// Verify runner registered in server
			runner := server.GetInspection(result.InspectionID)
			if runner == nil {
				t.Fatalf("registered inspection runner not found in server: %s", result.InspectionID)
			}
			if !runner.Started() {
				t.Errorf("runner.Started() should be true")
			}
		})
	}
}

func TestImportSessionManager_ParallelAndOutOfOrder(t *testing.T) {
	testCases := []struct {
		name               string
		inspectionName     string
		chunkSize          int
		wantInspectionName string
	}{
		{
			name:               "uploads chunks out of order",
			inspectionName:     "Cluster-OutOfOrder",
			chunkSize:          10,
			wantInspectionName: "Cluster-OutOfOrder",
		},
		{
			name:               "uploads chunks concurrently in parallel goroutines",
			inspectionName:     "Cluster-Concurrent",
			chunkSize:          8,
			wantInspectionName: "Cluster-Concurrent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			manager := NewImportSessionManager(server, ioConfig)
			defer manager.Close()

			data := createTestBinaryKHIBytes(t, tc.inspectionName)

			session, err := manager.StartSession("test.khi", int64(len(data)))
			if err != nil {
				t.Fatalf("StartSession failed: %v", err)
			}

			type chunkInfo struct {
				offset int64
				data   []byte
			}

			var chunks []chunkInfo
			for offset := 0; offset < len(data); offset += tc.chunkSize {
				end := offset + tc.chunkSize
				if end > len(data) {
					end = len(data)
				}
				chunks = append(chunks, chunkInfo{
					offset: int64(offset),
					data:   data[offset:end],
				})
			}

			if tc.name == "uploads chunks out of order" {
				// Upload in reverse order
				for i := len(chunks) - 1; i >= 0; i-- {
					c := chunks[i]
					if _, err := manager.WriteChunk(session.Token, c.offset, c.data); err != nil {
						t.Fatalf("WriteChunk failed at offset %d: %v", c.offset, err)
					}
				}
			} else {
				// Upload concurrently
				var wg sync.WaitGroup
				errChan := make(chan error, len(chunks))
				for _, c := range chunks {
					wg.Add(1)
					go func(info chunkInfo) {
						defer wg.Done()
						_, err := manager.WriteChunk(session.Token, info.offset, info.data)
						if err != nil {
							errChan <- err
						}
					}(c)
				}
				wg.Wait()
				close(errChan)
				for err := range errChan {
					t.Fatalf("concurrent WriteChunk error: %v", err)
				}
			}

			result, err := manager.CompleteSession(session.Token)
			if err != nil {
				t.Fatalf("CompleteSession failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantInspectionName, result.InspectionName); diff != "" {
				t.Errorf("InspectionName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestImportSessionManager_Errors(t *testing.T) {
	testCases := []struct {
		name      string
		run       func(t *testing.T, manager *ImportSessionManager) error
		wantErrIs error
	}{
		{
			name: "start session with non-positive total size",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				_, err := manager.StartSession("test.khi", 0)
				return err
			},
			wantErrIs: ErrInvalidTotalSize,
		},
		{
			name: "write chunk with non-existent token",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				_, err := manager.WriteChunk("non-existent-token", 0, []byte("data"))
				return err
			},
			wantErrIs: ErrSessionNotFound,
		},
		{
			name: "write chunk with empty data",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				_, err = manager.WriteChunk(session.Token, 0, []byte{})
				return err
			},
			wantErrIs: ErrEmptyChunkData,
		},
		{
			name: "write chunk with negative offset",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				_, err = manager.WriteChunk(session.Token, -1, []byte("data"))
				return err
			},
			wantErrIs: ErrInvalidOffset,
		},
		{
			name: "write chunk with offset exceeding total size",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				_, err = manager.WriteChunk(session.Token, 101, []byte("data"))
				return err
			},
			wantErrIs: ErrInvalidOffset,
		},
		{
			name: "write chunk exceeding maximum size",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				manager.chunkManager = chunkedupload.NewChunkSessionManager(t.TempDir(), chunkedupload.WithMaxChunkSize(10))
				session, err = manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				_, err = manager.WriteChunk(session.Token, 0, []byte("longer than 10 bytes"))
				return err
			},
			wantErrIs: ErrChunkSizeTooLarge,
		},
		{
			name: "complete session with missing chunks / gaps fails",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				// Upload byte 0-10, missing 10-100
				if _, err := manager.WriteChunk(session.Token, 0, []byte("0123456789")); err != nil {
					return err
				}
				_, err = manager.CompleteSession(session.Token)
				return err
			},
			wantErrIs: nil, // checked via non-nil error below
		},
		{
			name: "abort session cleans up temporary file",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				tempPath := session.TempFilePath
				if err := manager.AbortSession(session.Token); err != nil {
					return err
				}
				if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
					t.Errorf("temporary file was not deleted after abort: %s", tempPath)
				}
				// Subsequent abort should return ErrSessionNotFound
				return manager.AbortSession(session.Token)
			},
			wantErrIs: ErrSessionNotFound,
		},
		{
			name: "session expired and cleaned up returns ErrSessionNotFound",
			run: func(t *testing.T, manager *ImportSessionManager) error {
				session, err := manager.StartSession("test.khi", 100)
				if err != nil {
					return err
				}
				if err := manager.chunkManager.Evict(session.Token); err != nil {
					return err
				}
				_, err = manager.WriteChunk(session.Token, 0, []byte("data"))
				return err
			},
			wantErrIs: ErrSessionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			manager := NewImportSessionManager(server, ioConfig)
			defer manager.Close()

			err = tc.run(t, manager)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("error mismatch: got %v, want error matching %v", err, tc.wantErrIs)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		})
	}
}
