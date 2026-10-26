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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"google.golang.org/protobuf/proto"
)

func setupTestMaskServer(t *testing.T) (*httptest.Server, apiv1connect.WorkbenchServiceClient, apiv1connect.MaskServiceClient, *workbench.WorkbenchManager, string) {
	inspectionServer, validInspID := createTestInspectionServerForWorkbench(t)

	indexMgr := workbench.NewInspectionIndexManager(inspectionServer, t.TempDir())
	manager := workbench.NewWorkbenchManager(inspectionServer, indexMgr, 3)
	t.Cleanup(func() {
		if wb, err := manager.Get(validInspID, workbench.AccessorBrowser); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = wb.AwaitIndex(ctx)
		}
		indexMgr.Wait()
	})
	mux := http.NewServeMux()

	wbPath, wbHandler := apiv1connect.NewWorkbenchServiceHandler(NewWorkbenchServiceServer(manager))
	mux.Handle(wbPath, wbHandler)

	maskPath, maskHandler := apiv1connect.NewMaskServiceHandler(NewMaskServiceServer(manager))
	mux.Handle(maskPath, maskHandler)

	ts := httptest.NewServer(mux)
	wbClient := apiv1connect.NewWorkbenchServiceClient(ts.Client(), ts.URL)
	maskClient := apiv1connect.NewMaskServiceClient(ts.Client(), ts.URL)
	return ts, wbClient, maskClient, manager, validInspID
}

func TestMaskServiceServer_StringPoolIndexAndMasking(t *testing.T) {
	ts, wbClient, client, _, validInspID := setupTestMaskServer(t)
	defer ts.Close()

	openStream, err := wbClient.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() error = %v", err)
	}
	var wbID string
	for openStream.Receive() {
		if openStream.Msg().GetWorkbenchId() != "" {
			wbID = openStream.Msg().GetWorkbenchId()
		}
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream error = %v", err)
	}

	t.Run("WatchStringPoolIndexProgress streams to INDEX_STATE_READY", func(t *testing.T) {
		stream, err := client.WatchStringPoolIndexProgress(context.Background(), connect.NewRequest(&apiv1.WatchStringPoolIndexProgressRequest{
			WorkbenchId: proto.String(wbID),
		}))
		if err != nil {
			t.Fatalf("WatchStringPoolIndexProgress() error = %v", err)
		}
		var finalState apiv1.WatchIndexProgressResponse_IndexState
		for stream.Receive() {
			finalState = stream.Msg().GetState()
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("WatchStringPoolIndexProgress() stream error = %v", err)
		}
		if finalState != apiv1.WatchIndexProgressResponse_INDEX_STATE_READY {
			t.Errorf("final state = %v, want INDEX_STATE_READY", finalState)
		}
	})

	t.Run("PullStringPoolIndexProgress returns snapshot", func(t *testing.T) {
		resp, err := client.PullStringPoolIndexProgress(context.Background(), connect.NewRequest(&apiv1.PullStringPoolIndexProgressRequest{
			WorkbenchId: proto.String(wbID),
		}))
		if err != nil {
			t.Fatalf("PullStringPoolIndexProgress() error = %v", err)
		}
		if resp.Msg.GetState() != apiv1.WatchIndexProgressResponse_INDEX_STATE_READY {
			t.Errorf("PullStringPoolIndexProgress() state = %v, want INDEX_STATE_READY", resp.Msg.GetState())
		}
	})

	t.Run("ListWorkbenchStrings validates workbench_id, supports regex search, and rejects invalid regex", func(t *testing.T) {
		_, err := client.ListWorkbenchStrings(context.Background(), connect.NewRequest(&apiv1.ListWorkbenchStringsRequest{}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("ListWorkbenchStrings(empty workbench_id) code = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}

		resp, err := client.ListWorkbenchStrings(context.Background(), connect.NewRequest(&apiv1.ListWorkbenchStringsRequest{
			WorkbenchId: proto.String(wbID),
			Limit:       proto.Uint32(50),
		}))
		if err != nil {
			t.Fatalf("ListWorkbenchStrings() unexpected error = %v", err)
		}
		if resp.Msg.GetTotalCount() == 0 {
			t.Errorf("ListWorkbenchStrings() TotalCount = 0, want > 0")
		}

		regexResp, err := client.ListWorkbenchStrings(context.Background(), connect.NewRequest(&apiv1.ListWorkbenchStringsRequest{
			WorkbenchId: proto.String(wbID),
			Query:       proto.String(`^Test\s+Type$`),
			Limit:       proto.Uint32(50),
		}))
		if err != nil {
			t.Fatalf("ListWorkbenchStrings(regex query) unexpected error = %v", err)
		}
		if regexResp.Msg.GetTotalCount() != 1 {
			t.Errorf("ListWorkbenchStrings(regex query) TotalCount = %d, want 1", regexResp.Msg.GetTotalCount())
		}

		_, err = client.ListWorkbenchStrings(context.Background(), connect.NewRequest(&apiv1.ListWorkbenchStringsRequest{
			WorkbenchId: proto.String(wbID),
			Query:       proto.String(`[unclosed`),
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("ListWorkbenchStrings(invalid regex) code = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}
	})

	t.Run("CreateMaskedInspection streams progress frames and final result with sequential <masked_N> defaults and empty deletion", func(t *testing.T) {
		testCases := []struct {
			name        string
			req         *apiv1.CreateMaskedInspectionRequest
			wantErrCode connect.Code
			wantTitle   string
		}{
			{
				name: "missing rules fails with invalid argument",
				req: &apiv1.CreateMaskedInspectionRequest{
					WorkbenchId: proto.String(wbID),
				},
				wantErrCode: connect.CodeInvalidArgument,
			},
			{
				name: "unknown workbench_id fails with not found",
				req: &apiv1.CreateMaskedInspectionRequest{
					WorkbenchId: proto.String("non-existent-wb"),
					Rules: []*apiv1.StringMaskRule{
						{Target: proto.String("Test")},
					},
				},
				wantErrCode: connect.CodeNotFound,
			},
			{
				name: "unset replacements allocate non-colliding sequential <masked_N> placeholders",
				req: &apiv1.CreateMaskedInspectionRequest{
					WorkbenchId: proto.String(wbID),
					Rules: []*apiv1.StringMaskRule{
						{Target: proto.String("Test")},
						{Target: proto.String("other"), Replacement: proto.String("<masked_2>")},
						{Target: proto.String("Type")},
					},
				},
				wantErrCode: 0,
				wantTitle:   "<masked_1> <masked_3> (Masked)",
			},
			{
				name: "explicit empty replacement deletes substring",
				req: &apiv1.CreateMaskedInspectionRequest{
					WorkbenchId: proto.String(wbID),
					Rules: []*apiv1.StringMaskRule{
						{Target: proto.String("Test "), Replacement: proto.String("")},
					},
				},
				wantErrCode: 0,
				wantTitle:   "Type (Masked)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				stream, err := client.CreateMaskedInspection(context.Background(), connect.NewRequest(tc.req))
				if err != nil {
					if tc.wantErrCode != 0 {
						if connect.CodeOf(err) != tc.wantErrCode {
							t.Errorf("CreateMaskedInspection() error code = %v, want %v", connect.CodeOf(err), tc.wantErrCode)
						}
						return
					}
					t.Fatalf("CreateMaskedInspection() unexpected open error = %v", err)
				}

				var progressUpdates []float64
				var finalResult *apiv1.MaskedInspectionResult
				for stream.Receive() {
					msg := stream.Msg()
					if p := msg.GetProgress(); p != nil {
						progressUpdates = append(progressUpdates, p.GetProgressPercentage())
					}
					if r := msg.GetResult(); r != nil {
						finalResult = r
					}
				}
				streamErr := stream.Err()
				if tc.wantErrCode != 0 {
					if connect.CodeOf(streamErr) != tc.wantErrCode {
						t.Errorf("CreateMaskedInspection() stream error code = %v, want %v (err = %v)", connect.CodeOf(streamErr), tc.wantErrCode, streamErr)
					}
					return
				}
				if streamErr != nil {
					t.Fatalf("CreateMaskedInspection() unexpected stream error = %v", streamErr)
				}
				if len(progressUpdates) < 2 {
					t.Errorf("received %d progress updates, want at least 2", len(progressUpdates))
				}
				for i := 1; i < len(progressUpdates); i++ {
					if progressUpdates[i] < progressUpdates[i-1] {
						t.Errorf("progress decreased at index %d: %f < %f", i, progressUpdates[i], progressUpdates[i-1])
					}
				}
				if finalResult == nil {
					t.Fatalf("CreateMaskedInspection() ended without terminal result")
				}
				if finalResult.GetInspectionId() == "" {
					t.Errorf("CreateMaskedInspection() returned empty InspectionId")
				}
				if finalResult.GetInspectionName() != tc.wantTitle {
					t.Errorf("InspectionName = %q, want %q", finalResult.GetInspectionName(), tc.wantTitle)
				}
				if finalResult.GetFileSizeBytes() <= 0 {
					t.Errorf("FileSizeBytes = %d, want > 0", finalResult.GetFileSizeBytes())
				}
			})
		}
	})

	t.Run("CreateMaskedInspectionSync and CancelCreateMaskedInspectionSync support polling lifecycle", func(t *testing.T) {
		_, err := client.CreateMaskedInspectionSync(context.Background(), connect.NewRequest(&apiv1.CreateMaskedInspectionSyncRequest{
			WorkbenchId: proto.String(wbID),
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("CreateMaskedInspectionSync(empty rules) code = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}

		_, err = client.CancelCreateMaskedInspectionSync(context.Background(), connect.NewRequest(&apiv1.CancelCreateMaskedInspectionSyncRequest{
			WorkbenchId: proto.String(wbID),
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("CancelCreateMaskedInspectionSync(empty job_id) code = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}

		jobID := ""
		var finalResult *apiv1.MaskedInspectionResult
		for attempt := 0; attempt < 40; attempt++ {
			req := &apiv1.CreateMaskedInspectionSyncRequest{
				WorkbenchId: proto.String(wbID),
				JobId:       proto.String(jobID),
			}
			if jobID == "" {
				req.Rules = []*apiv1.StringMaskRule{
					{Target: proto.String("Test")},
				}
			}
			resp, pollErr := client.CreateMaskedInspectionSync(context.Background(), connect.NewRequest(req))
			if pollErr != nil {
				t.Fatalf("CreateMaskedInspectionSync() unexpected error = %v", pollErr)
			}
			jobID = resp.Msg.GetJobId()
			if jobID == "" {
				t.Fatalf("CreateMaskedInspectionSync() returned empty job_id")
			}
			if resp.Msg.GetIsDone() {
				if resp.Msg.GetErrorMessage() != "" {
					t.Fatalf("CreateMaskedInspectionSync() failed with error_message = %q", resp.Msg.GetErrorMessage())
				}
				finalResult = resp.Msg.GetResult()
				break
			}
			time.Sleep(25 * time.Millisecond)
		}

		if finalResult == nil {
			t.Fatalf("CreateMaskedInspectionSync() did not complete within polling budget")
		}
		if finalResult.GetInspectionName() != "<masked_1> Type (Masked)" {
			t.Errorf("InspectionName = %q, want %q", finalResult.GetInspectionName(), "<masked_1> Type (Masked)")
		}

		cancelResp, err := client.CancelCreateMaskedInspectionSync(context.Background(), connect.NewRequest(&apiv1.CancelCreateMaskedInspectionSyncRequest{
			WorkbenchId: proto.String(wbID),
			JobId:       proto.String(jobID),
		}))
		if err != nil {
			t.Fatalf("CancelCreateMaskedInspectionSync() unexpected error = %v", err)
		}
		if !cancelResp.Msg.GetCanceled() {
			t.Errorf("CancelCreateMaskedInspectionSync() Canceled = false, want true for retained job")
		}
	})
}
