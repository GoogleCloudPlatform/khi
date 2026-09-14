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
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/proto"
)

func createServerTestTask(refID, implHash string, deps []coretask.Dependency, opts ...coretask.LabelOpt) coretask.UntypedTask {
	ref := taskid.NewTaskReference[any](refID)
	id := taskid.NewImplementationID[any](ref, implHash)
	return coretask.NewTask[any](id, deps, func(ctx context.Context) (any, error) {
		return nil, nil
	}, opts...)
}

func TestInspectionTaskGraphServer_GetInspectionTaskRegistry(t *testing.T) {
	inspectionServer, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	err = inspectionServer.AddInspectionType(coreinspection.InspectionType{
		Id:          "gke-audit",
		Name:        "GKE Audit",
		Description: "GKE audit inspection type",
		Icon:        "gke-icon",
		Labels: map[string]string{
			"environment": "googlecloud",
		},
	})
	if err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	task := createServerTestTask("my.ref", "impl1", nil,
		coretask.WithSelectionPriority(10),
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"environment": "googlecloud"}),
	)
	if err := inspectionServer.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	server := NewInspectionTaskGraphServer(inspectionServer, DefaultStreamCycleDuration, DefaultUpdateInterval)

	testCases := []struct {
		name       string
		req        *apiv1.GetInspectionTaskRegistryRequest
		wantTypeID string
	}{
		{
			name:       "successfully retrieves registry",
			req:        &apiv1.GetInspectionTaskRegistryRequest{},
			wantTypeID: "gke-audit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := server.GetInspectionTaskRegistry(context.Background(), connect.NewRequest(tc.req))
			if err != nil {
				t.Fatalf("GetInspectionTaskRegistry() unexpected error = %v", err)
			}

			foundType := false
			for _, it := range res.Msg.GetInspectionTypes() {
				if it.GetId() == tc.wantTypeID {
					foundType = true
					break
				}
			}
			if !foundType {
				t.Errorf("GetInspectionTaskRegistry() expected inspection type %q in response", tc.wantTypeID)
			}

			foundGroup := false
			for _, g := range res.Msg.GetTaskGroups() {
				if g.GetTaskReferenceId() == "my.ref" {
					foundGroup = true
					if len(g.GetTasks()) != 1 {
						t.Errorf("len(g.Tasks) = %d, want 1", len(g.GetTasks()))
					}
					break
				}
			}
			if !foundGroup {
				t.Errorf("GetInspectionTaskRegistry() expected task group 'my.ref'")
			}
		})
	}
}

func TestInspectionTaskGraphServer_ResolveInspectionTaskGraph(t *testing.T) {
	inspectionServer, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	err = inspectionServer.AddInspectionType(coreinspection.InspectionType{
		Id:          "gke-audit",
		Name:        "GKE Audit",
		Description: "GKE audit inspection type",
		Labels: map[string]string{
			"environment": "googlecloud",
		},
	})
	if err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	taskA := createServerTestTask("feature.a", "impl1", nil,
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"environment": "googlecloud"}),
		inspectioncore_contract.FeatureTaskLabel("Feature A", "Description A", 1, true),
	)
	refA := taskid.NewTaskReference[any]("feature.a")
	taskB := createServerTestTask("task.b", "impl1", []coretask.Dependency{refA},
		inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"environment": "googlecloud"}),
		inspectioncore_contract.FeatureTaskLabel("Feature B", "Description B", 2, true),
	)

	if err := inspectionServer.AddTask(taskA); err != nil {
		t.Fatalf("failed to add taskA: %v", err)
	}
	if err := inspectionServer.AddTask(taskB); err != nil {
		t.Fatalf("failed to add taskB: %v", err)
	}

	server := NewInspectionTaskGraphServer(inspectionServer, DefaultStreamCycleDuration, DefaultUpdateInterval)

	testCases := []struct {
		name             string
		inspectionTypeID string
		featureOverrides map[string]bool
		wantCode         connect.Code
		wantSuccess      bool
	}{
		{
			name:             "empty inspection_type_id returns InvalidArgument",
			inspectionTypeID: "",
			wantCode:         connect.CodeInvalidArgument,
		},
		{
			name:             "non-existent inspection_type_id returns NotFound",
			inspectionTypeID: "non-existent-type",
			wantCode:         connect.CodeNotFound,
		},
		{
			name:             "valid inspection_type_id resolves DAG successfully",
			inspectionTypeID: "gke-audit",
			featureOverrides: map[string]bool{"feature.a#impl1": true, "task.b#impl1": true},
			wantCode:         0,
			wantSuccess:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &apiv1.ResolveInspectionTaskGraphRequest{
				InspectionTypeId: proto.String(tc.inspectionTypeID),
				FeatureOverrides: tc.featureOverrides,
			}
			res, err := server.ResolveInspectionTaskGraph(context.Background(), connect.NewRequest(req))
			if tc.wantCode != 0 {
				if err == nil {
					t.Fatalf("ResolveInspectionTaskGraph() expected error with code %v, got nil", tc.wantCode)
				}
				connectErr := new(connect.Error)
				if !errors.As(err, &connectErr) {
					t.Fatalf("ResolveInspectionTaskGraph() error is not *connect.Error: %v", err)
				}
				if connectErr.Code() != tc.wantCode {
					t.Errorf("ResolveInspectionTaskGraph() code = %v, want %v", connectErr.Code(), tc.wantCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveInspectionTaskGraph() unexpected error = %v", err)
			}

			if res.Msg.GetDag().GetIsSuccess() != tc.wantSuccess {
				t.Errorf("res.Dag.IsSuccess = %v, want %v", res.Msg.GetDag().GetIsSuccess(), tc.wantSuccess)
			}
			if len(res.Msg.GetFilteringEvaluations()) == 0 {
				t.Errorf("len(FilteringEvaluations) = 0, want > 0")
			}
			if len(res.Msg.GetAvailableFeatures()) == 0 {
				t.Errorf("len(AvailableFeatures) = 0, want > 0")
			}
		})
	}
}

// runTaskGraphFixture holds inspections in the states the run task graph handlers must distinguish.
type runTaskGraphFixture struct {
	inspectionServer       *coreinspection.InspectionTaskServer
	finishedInspectionID   string
	notStartedInspectionID string
}

// newInspectionServerWithTask initializes a test InspectionTaskServer with a single registered task and inspection type.
func newInspectionServerWithTask(t *testing.T, task coretask.UntypedTask) (*coreinspection.InspectionTaskServer, string) {
	t.Helper()
	logger.InitGlobalKHILogger()
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("NewIOConfigForTest failed: %v", err)
	}
	inspectionServer, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	inspectionType := coreinspection.InspectionType{Id: "test-inspection", Name: "Test Inspection"}
	if err := inspectionServer.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("AddInspectionType failed: %v", err)
	}
	if err := inspectionServer.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	return inspectionServer, inspectionType.Id
}

// newRunTaskGraphFixture builds an inspection server holding one finished inspection and one that
// was created but never started.
func newRunTaskGraphFixture(t *testing.T) runTaskGraphFixture {
	t.Helper()
	dummyTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("dummy-task"),
		nil,
		func(ctx context.Context) (any, error) {
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)
	inspectionServer, inspectionTypeID := newInspectionServerWithTask(t, dummyTask)

	finishedInspectionID, err := inspectionServer.CreateInspection(inspectionTypeID)
	if err != nil {
		t.Fatalf("CreateInspection failed: %v", err)
	}
	runner := inspectionServer.GetInspection(finishedInspectionID)
	if err := runner.Run(context.Background(), &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	<-runner.Wait()

	notStartedInspectionID, err := inspectionServer.CreateInspection(inspectionTypeID)
	if err != nil {
		t.Fatalf("CreateInspection failed: %v", err)
	}

	return runTaskGraphFixture{
		inspectionServer:       inspectionServer,
		finishedInspectionID:   finishedInspectionID,
		notStartedInspectionID: notStartedInspectionID,
	}
}

func TestInspectionTaskGraphServer_PullInspectionRunTaskGraph(t *testing.T) {
	fixture := newRunTaskGraphFixture(t)
	server := NewInspectionTaskGraphServer(fixture.inspectionServer, DefaultStreamCycleDuration, DefaultUpdateInterval)

	testCases := []struct {
		name         string
		inspectionID func(f runTaskGraphFixture) string
		wantCode     connect.Code
	}{
		{
			name:         "rejects an empty inspection ID",
			inspectionID: func(f runTaskGraphFixture) string { return "" },
			wantCode:     connect.CodeInvalidArgument,
		},
		{
			name:         "reports not found for an unregistered inspection",
			inspectionID: func(f runTaskGraphFixture) string { return "unknown-inspection-id" },
			wantCode:     connect.CodeNotFound,
		},
		{
			name:         "reports failed precondition before the task graph starts",
			inspectionID: func(f runTaskGraphFixture) string { return f.notStartedInspectionID },
			wantCode:     connect.CodeFailedPrecondition,
		},
		{
			name:         "returns the finished snapshot",
			inspectionID: func(f runTaskGraphFixture) string { return f.finishedInspectionID },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &apiv1.PullInspectionRunTaskGraphRequest{
				InspectionId: proto.String(tc.inspectionID(fixture)),
			}
			res, err := server.PullInspectionRunTaskGraph(context.Background(), connect.NewRequest(req))
			if tc.wantCode != 0 {
				if err == nil {
					t.Fatalf("PullInspectionRunTaskGraph() expected error with code %v, got nil", tc.wantCode)
				}
				connectErr := new(connect.Error)
				if !errors.As(err, &connectErr) {
					t.Fatalf("PullInspectionRunTaskGraph() error is not *connect.Error: %v", err)
				}
				if connectErr.Code() != tc.wantCode {
					t.Errorf("PullInspectionRunTaskGraph() code = %v, want %v", connectErr.Code(), tc.wantCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("PullInspectionRunTaskGraph() unexpected error = %v", err)
			}
			snapshot := res.Msg.GetSnapshot()
			if !snapshot.GetIsRunFinished() {
				t.Errorf("Snapshot.IsRunFinished = false, want true")
			}
			if !snapshot.GetDag().GetIsSuccess() {
				t.Errorf("Snapshot.Dag.IsSuccess = false, want true. error_message=%s", snapshot.GetDag().GetErrorMessage())
			}
			if len(snapshot.GetNodeStatuses()) != len(snapshot.GetDag().GetNodes()) {
				t.Errorf("len(NodeStatuses) = %d, want %d to cover every DAG node", len(snapshot.GetNodeStatuses()), len(snapshot.GetDag().GetNodes()))
			}
		})
	}
}

func TestInspectionTaskGraphServer_WatchInspectionRunTaskGraphClosesOnFinishedRun(t *testing.T) {
	fixture := newRunTaskGraphFixture(t)
	serverImpl := NewInspectionTaskGraphServer(fixture.inspectionServer, DefaultStreamCycleDuration, 10*time.Millisecond)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewInspectionTaskGraphServiceHandler(serverImpl)
	mux.Handle(path, handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := apiv1connect.NewInspectionTaskGraphServiceClient(ts.Client(), ts.URL)

	stream, err := client.WatchInspectionRunTaskGraph(context.Background(), connect.NewRequest(&apiv1.WatchInspectionRunTaskGraphRequest{
		InspectionId: proto.String(fixture.finishedInspectionID),
	}))
	if err != nil {
		t.Fatalf("WatchInspectionRunTaskGraph() failed: %v", err)
	}
	defer stream.Close()

	receivedCount := 0
	for stream.Receive() {
		receivedCount++
		snapshot := stream.Msg().GetSnapshot()
		if !snapshot.GetIsRunFinished() {
			t.Errorf("Snapshot.IsRunFinished = false, want true")
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err() = %v, want nil", err)
	}
	if receivedCount != 1 {
		t.Errorf("received %d snapshots, want 1 before the stream closes", receivedCount)
	}
}

func TestInspectionTaskGraphServer_WatchInspectionRunTaskGraphStreamsUntilTheRunFinishes(t *testing.T) {
	inspectionServer, inspectionID, releaseTask := newBlockingRunInspection(t)
	serverImpl := NewInspectionTaskGraphServer(inspectionServer, DefaultStreamCycleDuration, 10*time.Millisecond)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewInspectionTaskGraphServiceHandler(serverImpl)
	mux.Handle(path, handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := apiv1connect.NewInspectionTaskGraphServiceClient(ts.Client(), ts.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.WatchInspectionRunTaskGraph(ctx, connect.NewRequest(&apiv1.WatchInspectionRunTaskGraphRequest{
		InspectionId: proto.String(inspectionID),
	}))
	if err != nil {
		t.Fatalf("WatchInspectionRunTaskGraph() failed: %v", err)
	}
	defer stream.Close()

	receivedCount := 0
	unfinishedCount := 0
	lastIsRunFinished := false
	for stream.Receive() {
		receivedCount++
		lastIsRunFinished = stream.Msg().GetSnapshot().GetIsRunFinished()
		if lastIsRunFinished {
			continue
		}
		unfinishedCount++
		// Let the blocked task complete only after the stream proved it keeps emitting updates.
		if unfinishedCount == 2 {
			releaseTask()
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err() = %v, want nil", err)
	}
	if unfinishedCount < 2 {
		t.Errorf("received %d in-progress snapshots, want at least 2 to prove the update loop keeps running", unfinishedCount)
	}
	if !lastIsRunFinished {
		t.Errorf("last Snapshot.IsRunFinished = false, want true")
	}
	if receivedCount != unfinishedCount+1 {
		t.Errorf("received %d snapshots, want %d so the stream closes right after the finished one", receivedCount, unfinishedCount+1)
	}
}

func TestInspectionTaskGraphServer_WatchInspectionRunTaskGraphClosesAfterStreamCycle(t *testing.T) {
	inspectionServer, inspectionID, _ := newBlockingRunInspection(t)
	serverImpl := NewInspectionTaskGraphServer(inspectionServer, 50*time.Millisecond, 10*time.Millisecond)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewInspectionTaskGraphServiceHandler(serverImpl)
	mux.Handle(path, handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := apiv1connect.NewInspectionTaskGraphServiceClient(ts.Client(), ts.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.WatchInspectionRunTaskGraph(ctx, connect.NewRequest(&apiv1.WatchInspectionRunTaskGraphRequest{
		InspectionId: proto.String(inspectionID),
	}))
	if err != nil {
		t.Fatalf("WatchInspectionRunTaskGraph() failed: %v", err)
	}
	defer stream.Close()

	receivedCount := 0
	lastIsRunFinished := false
	for stream.Receive() {
		receivedCount++
		lastIsRunFinished = stream.Msg().GetSnapshot().GetIsRunFinished()
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream.Err() = %v, want nil after the stream cycle expires", err)
	}
	if lastIsRunFinished {
		t.Errorf("last Snapshot.IsRunFinished = true, want false because the run is still blocked")
	}
	if receivedCount < 2 {
		t.Errorf("received %d snapshots, want at least 2 before the cycle closes the stream", receivedCount)
	}
}

// newBlockingRunInspection starts an inspection whose only task blocks until the returned function
// is called, so tests can observe a run that is still in progress.
func newBlockingRunInspection(t *testing.T) (*coreinspection.InspectionTaskServer, string, func()) {
	t.Helper()
	release := make(chan struct{})
	blockingTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("blocking-task"),
		nil,
		func(ctx context.Context) (any, error) {
			select {
			case <-release:
				return "success", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)
	inspectionServer, inspectionTypeID := newInspectionServerWithTask(t, blockingTask)

	inspectionID, err := inspectionServer.CreateInspection(inspectionTypeID)
	if err != nil {
		t.Fatalf("CreateInspection failed: %v", err)
	}
	runner := inspectionServer.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var releaseOnce sync.Once
	releaseTask := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	t.Cleanup(func() {
		releaseTask()
		<-runner.Wait()
	})
	return inspectionServer, inspectionID, releaseTask
}
