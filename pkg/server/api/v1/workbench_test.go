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

	manager := workbench.NewWorkbenchManager(inspectionServer, 100*time.Millisecond, 0)
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
