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

package workbench

import (
	"context"
	"errors"
	"testing"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func createTestInspectionServer(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
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

func TestWorkbenchManager_GetOrOpen(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	testCases := []struct {
		name         string
		workbenchID  string
		inspectionID string
		wantErr      bool
	}{
		{
			name:         "opens new workbench successfully",
			workbenchID:  "user-session-1",
			inspectionID: validInspectionID,
			wantErr:      false,
		},
		{
			name:         "fails when inspection data not found",
			workbenchID:  "user-session-2",
			inspectionID: "invalid-inspection",
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewWorkbenchManager(inspectionServer, 100*time.Millisecond, 0)
			defer mgr.Stop()

			var progressEvents []apiv1.OpenWorkbenchResponse_Stage
			progressCb := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error {
				progressEvents = append(progressEvents, stage)
				return nil
			}

			wb, err := mgr.GetOrOpen(context.Background(), tc.workbenchID, tc.inspectionID, progressCb)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetOrOpen() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if wb.ID() != tc.workbenchID {
				t.Errorf("wb.ID() = %q, want %q", wb.ID(), tc.workbenchID)
			}

			if len(progressEvents) == 0 {
				t.Errorf("expected progress events to be captured")
			}
			if progressEvents[len(progressEvents)-1] != apiv1.OpenWorkbenchResponse_STAGE_READY {
				t.Errorf("final progress stage = %v, want STAGE_READY", progressEvents[len(progressEvents)-1])
			}
		})
	}
}

func TestWorkbenchManager_HeartbeatAndClose(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	mgr := NewWorkbenchManager(inspectionServer, 50*time.Millisecond, 0)
	defer mgr.Stop()

	wb, err := mgr.GetOrOpen(context.Background(), "user-session-1", validInspectionID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. Heartbeat succeeds
	_, expiresAt, err := mgr.Heartbeat(wb.ID())
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt %v should be after current time", expiresAt)
	}

	// 2. Heartbeat on unknown ID fails
	if _, _, err := mgr.Heartbeat("non-existent"); !errors.Is(err, ErrWorkbenchNotFound) {
		t.Errorf("Heartbeat() error = %v, want ErrWorkbenchNotFound", err)
	}

	// 3. Close frees session
	if err := mgr.Close(wb.ID()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := mgr.Get(wb.ID()); !errors.Is(err, ErrWorkbenchNotFound) {
		t.Errorf("Get() after Close() error = %v, want ErrWorkbenchNotFound", err)
	}
}

func TestWorkbenchManager_LeasesAndRemove(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	mgr := NewWorkbenchManager(inspectionServer, 50*time.Millisecond, 0)
	defer mgr.Stop()

	wb, err := mgr.GetOrOpen(context.Background(), "user-session-1", validInspectionID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	leases := mgr.Leases()
	expiresAt, ok := leases[wb.ID()]
	if !ok {
		t.Fatalf("expected leases to contain %q", wb.ID())
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expected lease expiration to be in the future, got %v", expiresAt)
	}

	// Remove frees session
	mgr.Remove(wb.ID())
	if _, err := mgr.Get(wb.ID()); !errors.Is(err, ErrWorkbenchNotFound) {
		t.Errorf("Get() after Remove() error = %v, want ErrWorkbenchNotFound", err)
	}
	if _, ok := mgr.Leases()[wb.ID()]; ok {
		t.Errorf("expected lease for %q to be deleted after Remove()", wb.ID())
	}
}
