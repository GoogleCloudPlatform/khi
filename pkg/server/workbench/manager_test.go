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
	"sync"
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

	noopProgress := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error { return nil }

	wb, err := mgr.GetOrOpen(context.Background(), "user-session-1", validInspectionID, noopProgress)
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

	noopProgress := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error { return nil }

	wb, err := mgr.GetOrOpen(context.Background(), "user-session-1", validInspectionID, noopProgress)
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

func TestWorkbenchManager_GetAndTouch(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	mgr := NewWorkbenchManager(inspectionServer, 50*time.Millisecond, 0)
	defer mgr.Stop()

	noopProgress := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error { return nil }

	wb, err := mgr.GetOrOpen(context.Background(), "user-session-1", validInspectionID, noopProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testCases := []struct {
		name        string
		workbenchID string
		wantErrIs   error
	}{
		{
			name:        "successfully gets workbench and refreshes TTL",
			workbenchID: wb.ID(),
			wantErrIs:   nil,
		},
		{
			name:        "returns ErrWorkbenchNotFound for non-existent ID",
			workbenchID: "unknown-session",
			wantErrIs:   ErrWorkbenchNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotWb, err := mgr.GetAndTouch(tc.workbenchID)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("GetAndTouch(%q) error = %v, want %v", tc.workbenchID, err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetAndTouch(%q) unexpected error = %v", tc.workbenchID, err)
			}
			if gotWb.ID() != tc.workbenchID {
				t.Errorf("GetAndTouch(%q) ID = %q, want %q", tc.workbenchID, gotWb.ID(), tc.workbenchID)
			}
		})
	}
}

func TestWorkbenchManager_ReopenDifferentInspection(t *testing.T) {
	inspectionServer, validInspectionID1 := createTestInspectionServer(t)

	// Create second inspection
	validInspectionID2, err := inspectionServer.CreateInspection("test-type")
	if err != nil {
		t.Fatalf("failed to create second inspection: %v", err)
	}
	runner := inspectionServer.GetInspection(validInspectionID2)
	if err := runner.Run(context.Background(), &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run second inspection: %v", err)
	}
	<-runner.Wait()

	mgr := NewWorkbenchManager(inspectionServer, 5*time.Second, 0)
	defer mgr.Stop()

	workbenchID := "user-session-same"

	noopProgress := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error { return nil }

	// 1. Open inspection 1
	wb1, err := mgr.GetOrOpen(context.Background(), workbenchID, validInspectionID1, noopProgress)
	if err != nil {
		t.Fatalf("GetOrOpen(inspection1) unexpected error: %v", err)
	}
	if wb1.InspectionID() != validInspectionID1 {
		t.Errorf("wb1.InspectionID() = %q, want %q", wb1.InspectionID(), validInspectionID1)
	}

	// 2. Open inspection 2 with the SAME workbench ID
	var progressStages []apiv1.OpenWorkbenchResponse_Stage
	progressCb := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error {
		progressStages = append(progressStages, stage)
		return nil
	}

	wb2, err := mgr.GetOrOpen(context.Background(), workbenchID, validInspectionID2, progressCb)
	if err != nil {
		t.Fatalf("GetOrOpen(inspection2) unexpected error: %v", err)
	}

	// Verify old workbench is closed
	if !wb1.IsClosed() {
		t.Errorf("expected wb1 to be closed after opening different inspection")
	}

	// Verify new workbench has inspection 2
	if wb2.InspectionID() != validInspectionID2 {
		t.Errorf("wb2.InspectionID() = %q, want %q", wb2.InspectionID(), validInspectionID2)
	}

	// Verify full progress lifecycle was executed for inspection 2 (not just STAGE_READY attached)
	if len(progressStages) < 2 {
		t.Errorf("expected full progress events for new inspection, got: %v", progressStages)
	}
	if progressStages[0] != apiv1.OpenWorkbenchResponse_STAGE_INITIALIZING {
		t.Errorf("first progress stage = %v, want STAGE_INITIALIZING", progressStages[0])
	}
}

func TestWorkbenchManager_ConcurrentGetOrOpen(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	mgr := NewWorkbenchManager(inspectionServer, 5*time.Second, 0)
	defer mgr.Stop()

	const numConcurrent = 10
	workbenchID := "concurrent-user-session"

	results := make([]*Workbench, numConcurrent)
	errs := make([]error, numConcurrent)

	var wg sync.WaitGroup
	wg.Add(numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		workerIdx := i
		go func() {
			defer wg.Done()
			wb, err := mgr.GetOrOpen(context.Background(), workbenchID, validInspectionID, nil)
			results[workerIdx] = wb
			errs[workerIdx] = err
		}()
	}

	wg.Wait()

	for i := 0; i < numConcurrent; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d returned error: %v", i, errs[i])
		}
		if results[i] == nil {
			t.Fatalf("goroutine %d returned nil Workbench", i)
		}
		if results[i] != results[0] {
			t.Errorf("goroutine %d returned workbench instance %p, want %p", i, results[i], results[0])
		}
	}
}
