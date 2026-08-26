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
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/proto"
)

func createTestInspectionServerForWorkbench(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
	logger.InitGlobalKHILogger()
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create test IOConfig: %v", err)
	}
	tempDir := t.TempDir()
	ioConfig.DataDestination = tempDir
	ioConfig.TemporaryFolder = tempDir
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	inspectionType := coreinspection.InspectionType{
		Id:   "test-type",
		Name: "Test Type",
	}
	if err := server.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	dummyTask := coretask.NewTask(
		dummyTaskID,
		nil,
		func(ctx context.Context) (any, error) {
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionTypes, []string{inspectionType.Id}),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)
	if err := server.AddTask(dummyTask); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	inspectionID, err := server.CreateInspection(inspectionType.Id)
	if err != nil {
		t.Fatalf("failed to create inspection: %v", err)
	}

	runner := server.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run inspection: %v", err)
	}
	<-runner.Wait()

	return server, inspectionID
}

func setupTestWorkbenchServer(t *testing.T) (*httptest.Server, apiv1connect.WorkbenchServiceClient, *workbench.WorkbenchManager, string) {
	inspectionServer, validInspID := createTestInspectionServerForWorkbench(t)

	indexMgr := workbench.NewInspectionIndexManager(inspectionServer, t.TempDir())
	manager := workbench.NewWorkbenchManager(inspectionServer, indexMgr, 100*time.Millisecond, 0)
	serverImpl := NewWorkbenchServiceServer(manager)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewWorkbenchServiceHandler(serverImpl)
	mux.Handle(path, handler)

	ts := httptest.NewServer(mux)
	client := apiv1connect.NewWorkbenchServiceClient(ts.Client(), ts.URL)
	return ts, client, manager, validInspID
}

func TestWorkbenchServiceServer_OpenWorkbench(t *testing.T) {
	ts, client, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	testCases := []struct {
		name        string
		req         *apiv1.OpenWorkbenchRequest
		wantErrCode connect.Code
	}{
		{
			name: "successfully opens workbench and streams stages",
			req: &apiv1.OpenWorkbenchRequest{
				UserId:       proto.String("user-1"),
				SessionId:    proto.String("session-0"),
				InspectionId: proto.String(validInspID),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid arguments when missing parameters",
			req: &apiv1.OpenWorkbenchRequest{
				UserId: proto.String("user-1"),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name: "fails when inspection dataset not found",
			req: &apiv1.OpenWorkbenchRequest{
				UserId:       proto.String("user-1"),
				SessionId:    proto.String("session-1"),
				InspectionId: proto.String("unknown-insp"),
			},
			wantErrCode: connect.CodeInternal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(tc.req))
			if err != nil {
				t.Fatalf("OpenWorkbench() error = %v", err)
			}

			var responses []*apiv1.OpenWorkbenchResponse
			var streamErr error
			for stream.Receive() {
				responses = append(responses, stream.Msg())
			}
			streamErr = stream.Err()

			if tc.wantErrCode != 0 {
				if streamErr == nil {
					t.Fatalf("expected stream error with code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(streamErr) != tc.wantErrCode {
					t.Errorf("stream error code = %v, want %v (err: %v)", connect.CodeOf(streamErr), tc.wantErrCode, streamErr)
				}
				return
			}

			if streamErr != nil {
				t.Fatalf("unexpected stream error: %v", streamErr)
			}

			if len(responses) == 0 {
				t.Fatalf("expected streamed responses, got 0")
			}

			finalRes := responses[len(responses)-1]
			if finalRes.GetStage() != apiv1.OpenWorkbenchResponse_STAGE_READY {
				t.Errorf("final stage = %v, want STAGE_READY", finalRes.GetStage())
			}

			if finalRes.GetWorkbenchId() != "user-1-session-0" {
				t.Errorf("WorkbenchId = %q, want %q", finalRes.GetWorkbenchId(), "user-1-session-0")
			}
		})
	}
}

func TestWorkbenchServiceServer_HeartbeatAndClose(t *testing.T) {
	ts, client, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	// 1. Open workbench first
	openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		UserId:       proto.String("user-hb"),
		SessionId:    proto.String("session-0"),
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() error = %v", err)
	}
	for openStream.Receive() {
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream error = %v", err)
	}

	workbenchID := "user-hb-session-0"

	// 2. Heartbeat on active workbench
	hbRes, err := client.HeartbeatWorkbench(context.Background(), connect.NewRequest(&apiv1.HeartbeatWorkbenchRequest{
		WorkbenchId: proto.String(workbenchID),
	}))
	if err != nil {
		t.Fatalf("HeartbeatWorkbench() unexpected error: %v", err)
	}
	if !hbRes.Msg.GetActive() {
		t.Errorf("HeartbeatWorkbench() active = false, want true")
	}

	// 3. Close workbench
	closeRes, err := client.CloseWorkbench(context.Background(), connect.NewRequest(&apiv1.CloseWorkbenchRequest{
		WorkbenchId: proto.String(workbenchID),
	}))
	if err != nil {
		t.Fatalf("CloseWorkbench() unexpected error: %v", err)
	}
	if !closeRes.Msg.GetClosed() {
		t.Errorf("CloseWorkbench() closed = false, want true")
	}

	// 4. Heartbeat on closed workbench returns NotFound
	_, err = client.HeartbeatWorkbench(context.Background(), connect.NewRequest(&apiv1.HeartbeatWorkbenchRequest{
		WorkbenchId: proto.String(workbenchID),
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("HeartbeatWorkbench() after close code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestWorkbenchServiceServer_ReadStructYAML(t *testing.T) {
	ts, client, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	// 1. Open workbench
	openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		UserId:       proto.String("user-struct"),
		SessionId:    proto.String("session-0"),
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() error = %v", err)
	}
	for openStream.Receive() {
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream error = %v", err)
	}

	workbenchID := "user-struct-session-0"

	testCases := []struct {
		name        string
		workbenchID string
		structID    uint32
		wantErrCode connect.Code
	}{
		{
			name:        "fails with invalid argument when workbench_id is empty",
			workbenchID: "",
			structID:    1,
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name:        "fails with invalid argument when struct_id is 0",
			workbenchID: workbenchID,
			structID:    0,
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name:        "fails with not found for non-existent workbench",
			workbenchID: "non-existent-workbench",
			structID:    1,
			wantErrCode: connect.CodeNotFound,
		},
		{
			name:        "fails with not found for non-existent struct ID",
			workbenchID: workbenchID,
			structID:    9999,
			wantErrCode: connect.CodeNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := client.ReadStructYAML(context.Background(), connect.NewRequest(&apiv1.ReadStructYAMLRequest{
				WorkbenchId: proto.String(tc.workbenchID),
				StructId:    proto.Uint32(tc.structID),
			}))
			if tc.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(err) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v (err = %v)", connect.CodeOf(err), tc.wantErrCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Msg.GetYaml() == "" {
				t.Errorf("expected non-empty YAML response")
			}
		})
	}
}

func TestWorkbenchServiceServer_FilterTimeline(t *testing.T) {
	ts, client, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	// 1. Open workbench
	openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		UserId:       proto.String("user-filter"),
		SessionId:    proto.String("session-0"),
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() error = %v", err)
	}
	for openStream.Receive() {
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream error = %v", err)
	}

	workbenchID := "user-filter-session-0"

	testCases := []struct {
		name        string
		req         *apiv1.FilterTimelineRequest
		wantErrCode connect.Code
	}{
		{
			name: "successfully filters timeline with streaming progress",
			req: &apiv1.FilterTimelineRequest{
				WorkbenchId:   proto.String(workbenchID),
				TimelineQuery: proto.String(""),
				LogQuery:      proto.String(""),
				ExcludeNoLogs: proto.Bool(false),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid argument when workbench_id is empty",
			req: &apiv1.FilterTimelineRequest{
				WorkbenchId: proto.String(""),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name: "fails with not found for non-existent workbench",
			req: &apiv1.FilterTimelineRequest{
				WorkbenchId: proto.String("non-existent-wb"),
			},
			wantErrCode: connect.CodeNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stream, err := client.FilterTimeline(context.Background(), connect.NewRequest(tc.req))
			if err != nil {
				t.Fatalf("FilterTimeline() error = %v", err)
			}

			var responses []*apiv1.FilterTimelineResponse
			for stream.Receive() {
				responses = append(responses, stream.Msg())
			}
			streamErr := stream.Err()

			if tc.wantErrCode != 0 {
				if streamErr == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(streamErr) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v (err = %v)", connect.CodeOf(streamErr), tc.wantErrCode, streamErr)
				}
				return
			}

			if streamErr != nil {
				t.Fatalf("unexpected stream error: %v", streamErr)
			}

			if len(responses) == 0 {
				t.Fatalf("expected streamed responses, got 0")
			}

			finalRes := responses[len(responses)-1]
			if finalRes.GetResult() == nil {
				t.Errorf("final response does not contain Result")
			}
		})
	}
}

func TestWorkbenchServiceServer_WatchIndexProgress(t *testing.T) {
	ts, client, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	// 1. Open workbench first
	openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		UserId:       proto.String("user-watch"),
		SessionId:    proto.String("session-0"),
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() error = %v", err)
	}
	for openStream.Receive() {
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream error = %v", err)
	}

	workbenchID := "user-watch-session-0"

	testCases := []struct {
		name        string
		req         *apiv1.WatchIndexProgressRequest
		wantErrCode connect.Code
	}{
		{
			name: "success on valid workbench id",
			req: &apiv1.WatchIndexProgressRequest{
				WorkbenchId: proto.String(workbenchID),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid argument when workbench_id is empty",
			req: &apiv1.WatchIndexProgressRequest{
				WorkbenchId: proto.String(""),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name: "fails with not found for non-existent workbench",
			req: &apiv1.WatchIndexProgressRequest{
				WorkbenchId: proto.String("non-existent-wb"),
			},
			wantErrCode: connect.CodeNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			stream, err := client.WatchIndexProgress(ctx, connect.NewRequest(tc.req))
			if err != nil {
				t.Fatalf("WatchIndexProgress() error = %v", err)
			}

			var responses []*apiv1.WatchIndexProgressResponse
			for stream.Receive() {
				responses = append(responses, stream.Msg())
			}
			streamErr := stream.Err()

			if tc.wantErrCode != 0 {
				if streamErr == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(streamErr) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v (err = %v)", connect.CodeOf(streamErr), tc.wantErrCode, streamErr)
				}
				return
			}

			if streamErr != nil {
				t.Fatalf("unexpected stream error: %v", streamErr)
			}

			if len(responses) == 0 {
				t.Fatalf("expected streamed responses, got 0")
			}
		})
	}
}

func TestWorkbenchServiceServer_ProtoJSONClient(t *testing.T) {
	ts, _, manager, validInspID := setupTestWorkbenchServer(t)
	defer ts.Close()
	defer manager.Stop()

	// Client configured with connect.WithProtoJSON() to mimic frontend development mode.
	jsonClient := apiv1connect.NewWorkbenchServiceClient(ts.Client(), ts.URL, connect.WithProtoJSON())

	openStream, err := jsonClient.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
		UserId:       proto.String("user-json"),
		SessionId:    proto.String("session-0"),
		InspectionId: proto.String(validInspID),
	}))
	if err != nil {
		t.Fatalf("OpenWorkbench() with ProtoJSON error = %v", err)
	}
	var lastMsg *apiv1.OpenWorkbenchResponse
	for openStream.Receive() {
		lastMsg = openStream.Msg()
	}
	if err := openStream.Err(); err != nil {
		t.Fatalf("OpenWorkbench() stream with ProtoJSON error = %v", err)
	}
	if lastMsg.GetWorkbenchId() == "" {
		t.Fatalf("expected last OpenWorkbench message to have workbench_id, got %v", lastMsg)
	}

	hbRes, err := jsonClient.HeartbeatWorkbench(context.Background(), connect.NewRequest(&apiv1.HeartbeatWorkbenchRequest{
		WorkbenchId: proto.String(lastMsg.GetWorkbenchId()),
	}))
	if err != nil {
		t.Fatalf("HeartbeatWorkbench() with ProtoJSON error = %v", err)
	}
	if !hbRes.Msg.GetActive() {
		t.Errorf("HeartbeatWorkbench() active = false, want true")
	}
}

func TestWorkbenchServiceServer_OpenWorkbenchSync_And_Cancel(t *testing.T) {
	testCases := []struct {
		name        string
		req         *apiv1.OpenWorkbenchSyncRequest
		wantErrCode connect.Code
	}{
		{
			name: "opens workbench synchronously and completes",
			req: &apiv1.OpenWorkbenchSyncRequest{
				UserId:    proto.String("user-sync"),
				SessionId: proto.String("session-sync"),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid argument when parameters missing",
			req: &apiv1.OpenWorkbenchSyncRequest{
				UserId: proto.String(""),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, manager, validInspID := setupTestWorkbenchServer(t)
			defer ts.Close()
			defer manager.Stop()

			if tc.wantErrCode == 0 {
				tc.req.InspectionId = proto.String(validInspID)
			}

			res, err := client.OpenWorkbenchSync(context.Background(), connect.NewRequest(tc.req))
			if tc.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(err) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v", connect.CodeOf(err), tc.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("OpenWorkbenchSync() unexpected error = %v", err)
			}

			// If not done yet, poll until done
			jobID := res.Msg.GetJobId()
			for res.Msg.GetStage() != apiv1.OpenWorkbenchResponse_STAGE_READY {
				pollRes, err := client.OpenWorkbenchSync(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchSyncRequest{
					JobId: proto.String(jobID),
				}))
				if err != nil {
					t.Fatalf("polling OpenWorkbenchSync() error = %v", err)
				}
				res = pollRes
				time.Sleep(10 * time.Millisecond)
			}

			if res.Msg.GetWorkbenchId() == "" {
				t.Errorf("expected non-empty workbench ID on completion")
			}
		})
	}
}

func TestWorkbenchServiceServer_PullIndexProgress(t *testing.T) {
	testCases := []struct {
		name        string
		req         *apiv1.PullIndexProgressRequest
		wantErrCode connect.Code
	}{
		{
			name: "pulls index progress successfully",
			req: &apiv1.PullIndexProgressRequest{
				WorkbenchId: proto.String("user-pull-idx-session-0"),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid argument when workbench ID is empty",
			req: &apiv1.PullIndexProgressRequest{
				WorkbenchId: proto.String(""),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
		{
			name: "fails with not found for unknown workbench",
			req: &apiv1.PullIndexProgressRequest{
				WorkbenchId: proto.String("unknown-wb"),
			},
			wantErrCode: connect.CodeNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, manager, validInspID := setupTestWorkbenchServer(t)
			defer ts.Close()
			defer manager.Stop()

			if tc.wantErrCode == 0 {
				// Open workbench first
				openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
					UserId:       proto.String("user-pull-idx"),
					SessionId:    proto.String("session-0"),
					InspectionId: proto.String(validInspID),
				}))
				if err != nil {
					t.Fatalf("OpenWorkbench() error = %v", err)
				}
				for openStream.Receive() {
				}
				if err := openStream.Err(); err != nil {
					t.Fatalf("OpenWorkbench() stream error = %v", err)
				}
			}

			res, err := client.PullIndexProgress(context.Background(), connect.NewRequest(tc.req))
			if tc.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(err) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v", connect.CodeOf(err), tc.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("PullIndexProgress() unexpected error = %v", err)
			}
			if res.Msg.GetState() == apiv1.WatchIndexProgressResponse_INDEX_STATE_UNSPECIFIED {
				t.Errorf("unexpected index state: %v", res.Msg.GetState())
			}
		})
	}
}

func TestWorkbenchServiceServer_FilterTimelineSync_And_Cancel(t *testing.T) {
	testCases := []struct {
		name        string
		req         *apiv1.FilterTimelineSyncRequest
		wantErrCode connect.Code
	}{
		{
			name: "filters timeline synchronously",
			req: &apiv1.FilterTimelineSyncRequest{
				WorkbenchId: proto.String("user-filter-sync-session-0"),
			},
			wantErrCode: 0,
		},
		{
			name: "fails with invalid argument when workbench ID is empty",
			req: &apiv1.FilterTimelineSyncRequest{
				WorkbenchId: proto.String(""),
			},
			wantErrCode: connect.CodeInvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, manager, validInspID := setupTestWorkbenchServer(t)
			defer ts.Close()
			defer manager.Stop()

			if tc.wantErrCode == 0 {
				// Open workbench first
				openStream, err := client.OpenWorkbench(context.Background(), connect.NewRequest(&apiv1.OpenWorkbenchRequest{
					UserId:       proto.String("user-filter-sync"),
					SessionId:    proto.String("session-0"),
					InspectionId: proto.String(validInspID),
				}))
				if err != nil {
					t.Fatalf("OpenWorkbench() error = %v", err)
				}
				for openStream.Receive() {
				}
				if err := openStream.Err(); err != nil {
					t.Fatalf("OpenWorkbench() stream error = %v", err)
				}
			}

			res, err := client.FilterTimelineSync(context.Background(), connect.NewRequest(tc.req))
			if tc.wantErrCode != 0 {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tc.wantErrCode)
				}
				if connect.CodeOf(err) != tc.wantErrCode {
					t.Errorf("error code = %v, want %v", connect.CodeOf(err), tc.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("FilterTimelineSync() unexpected error = %v", err)
			}

			jobID := res.Msg.GetJobId()
			for !res.Msg.GetIsDone() {
				pollRes, err := client.FilterTimelineSync(context.Background(), connect.NewRequest(&apiv1.FilterTimelineSyncRequest{
					WorkbenchId: tc.req.WorkbenchId,
					JobId:       proto.String(jobID),
				}))
				if err != nil {
					t.Fatalf("polling FilterTimelineSync() error = %v", err)
				}
				res = pollRes
				time.Sleep(10 * time.Millisecond)
			}

			if res.Msg.GetResult() == nil {
				t.Errorf("expected non-nil result on filter completion")
			}
		})
	}
}
