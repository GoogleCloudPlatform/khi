// Copyright 2025 Google LLC
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

package coreinspection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInspectionTaskRunner_Interceptor(t *testing.T) {
	// Initialize global logger
	logger.InitGlobalKHILogger()

	// Setup minimal server
	ioConfig := &inspectioncore_contract.IOConfig{
		TemporaryFolder: t.TempDir(),
	}
	server, err := NewServer(ioConfig)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	inspectionType := InspectionType{
		Id:   "test-inspection",
		Name: "Test Inspection",
	}
	if err := server.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("AddInspectionType failed: %v", err)
	}

	// Add a dummy task that is enabled for this inspection type
	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	dummyTask := coretask.NewTask(
		dummyTaskID,
		nil,
		func(ctx context.Context) (any, error) {
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)
	if err := server.AddTask(dummyTask); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Create inspection
	inspectionID, err := server.CreateInspection(inspectionType.Id)
	if err != nil {
		t.Fatalf("CreateInspection failed: %v", err)
	}
	runner := server.GetInspection(inspectionID)

	// Add interceptors
	executionOrder := []string{}
	interceptor1 := func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		executionOrder = append(executionOrder, "interceptor1_start")
		err := next(ctx)
		executionOrder = append(executionOrder, "interceptor1_end")
		return err
	}
	interceptor2 := func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		executionOrder = append(executionOrder, "interceptor2_start")
		err := next(ctx)
		executionOrder = append(executionOrder, "interceptor2_end")
		return err
	}

	runner.AddInterceptors(interceptor1, interceptor2)

	// Run inspection
	req := &inspectioncore_contract.InspectionRequest{
		Values: map[string]any{},
	}
	err = runner.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	<-runner.Wait()

	expectedOrder := []string{
		"interceptor1_start",
		"interceptor2_start",
		"interceptor2_end",
		"interceptor1_end",
	}

	if diff := cmp.Diff(expectedOrder, executionOrder); diff != "" {
		t.Errorf("Execution order mismatch (-want +got):\n%s", diff)
	}
}

func TestIsTaskCompatible(t *testing.T) {
	tests := []struct {
		name           string
		labelOpts      []coretask.LabelOpt
		inspectionType *InspectionType
		want           bool
	}{
		{
			name: "Selector matches target labels",
			labelOpts: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(inspectioncore_contract.LabelSelector{"platform": "gke"}),
			},
			inspectionType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gke", "provider": "google"},
			},
			want: true,
		},
		{
			name: "Selector does not match target labels due to value mismatch",
			labelOpts: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(inspectioncore_contract.LabelSelector{"platform": "gke"}),
			},
			inspectionType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gdc"},
			},
			want: false,
		},
		{
			name: "Selector does not match target labels due to missing key in target",
			labelOpts: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(inspectioncore_contract.LabelSelector{"platform": "gke"}),
			},
			inspectionType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"provider": "google"},
			},
			want: false,
		},
		{
			name: "Multi-key selector matches when all keys match",
			labelOpts: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(inspectioncore_contract.LabelSelector{
					"platform": "gke",
					"provider": "google",
				}),
			},
			inspectionType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gke", "provider": "google", "region": "us-central1"},
			},
			want: true,
		},
		{
			name: "Multi-key selector fails when only some keys match",
			labelOpts: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(inspectioncore_contract.LabelSelector{
					"platform": "gke",
					"provider": "aws",
				}),
			},
			inspectionType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gke", "provider": "google"},
			},
			want: false,
		},
		{
			name:      "No selector (Global task)",
			labelOpts: []coretask.LabelOpt{},
			inspectionType: &InspectionType{
				Id: "any-env",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := coretask.NewTask(
				taskid.NewDefaultImplementationID[any]("test-task"),
				nil,
				func(ctx context.Context) (any, error) { return nil, nil },
				tt.labelOpts...,
			)

			got, _ := EvaluateTaskCompatibility(task, tt.inspectionType)
			if got != tt.want {
				t.Errorf("EvaluateTaskCompatibility() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeduplicateTasksByPriority(t *testing.T) {
	taskRefA := taskid.NewTaskReference[any]("task-a")
	taskRefB := taskid.NewTaskReference[any]("task-b")

	taskAImpl1 := coretask.NewTask(
		taskid.NewImplementationID(taskRefA, "impl1"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(10),
	)
	taskAImpl2 := coretask.NewTask(
		taskid.NewImplementationID(taskRefA, "impl2"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(100),
	)
	taskAImpl3 := coretask.NewTask(
		taskid.NewImplementationID(taskRefA, "impl3"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(50),
	)
	taskBImpl1 := coretask.NewTask(
		taskid.NewImplementationID(taskRefB, "impl1"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(0),
	)
	taskATie1 := coretask.NewTask(
		taskid.NewImplementationID(taskRefA, "tie-a"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(20),
	)
	taskATie2 := coretask.NewTask(
		taskid.NewImplementationID(taskRefA, "tie-b"),
		nil,
		func(ctx context.Context) (any, error) { return nil, nil },
		coretask.WithSelectionPriority(20),
	)

	tests := []struct {
		name    string
		tasks   []coretask.UntypedTask
		wantIDs []string
	}{
		{
			name:    "single task remains unchanged",
			tasks:   []coretask.UntypedTask{taskAImpl1},
			wantIDs: []string{"task-a#impl1"},
		},
		{
			name:    "distinct TaskRefs are all retained",
			tasks:   []coretask.UntypedTask{taskAImpl1, taskBImpl1},
			wantIDs: []string{"task-a#impl1", "task-b#impl1"},
		},
		{
			name:    "same TaskRef selects highest priority task",
			tasks:   []coretask.UntypedTask{taskAImpl1, taskAImpl2, taskAImpl3},
			wantIDs: []string{"task-a#impl2"},
		},
		{
			name:    "multiple TaskRefs select respective highest priority tasks",
			tasks:   []coretask.UntypedTask{taskAImpl1, taskAImpl2, taskBImpl1},
			wantIDs: []string{"task-a#impl2", "task-b#impl1"},
		},
		{
			name:    "same priority tie-break deterministically chooses higher string ID",
			tasks:   []coretask.UntypedTask{taskATie1, taskATie2},
			wantIDs: []string{"task-a#tie-b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTasks := deduplicateTasksByPriority(tc.tasks)
			gotIDs := make([]string, 0, len(gotTasks))
			for _, task := range gotTasks {
				gotIDs = append(gotIDs, task.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantIDs, gotIDs, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("deduplicateTasksByPriority() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetInspectionType_SelectionPriority(t *testing.T) {
	tests := []struct {
		name                 string
		tasks                []coretask.UntypedTask
		inspectionType       InspectionType
		wantAvailableTaskIDs []string
		wantFeatureListIDs   []string
		wantEnabledFeatures  []string
	}{
		{
			name: "selects specialized task with higher priority over generic task",
			inspectionType: InspectionType{
				Id:     "gke-inspection",
				Name:   "GKE Inspection",
				Labels: map[string]string{"platform": "gke"},
			},
			tasks: []coretask.UntypedTask{
				// Generic implementation with priority 0, default feature
				coretask.NewTask(
					taskid.NewImplementationID(taskid.NewTaskReference[any]("audit-log-parser"), "generic"),
					nil,
					func(ctx context.Context) (any, error) { return nil, nil },
					inspectioncore_contract.FeatureTaskLabel("Generic Audit Logs", "Generic description", 100, true),
					coretask.WithSelectionPriority(0),
				),
				// GKE specialized implementation with priority 100, default feature
				coretask.NewTask(
					taskid.NewImplementationID(taskid.NewTaskReference[any]("audit-log-parser"), "gke"),
					nil,
					func(ctx context.Context) (any, error) { return nil, nil },
					inspectioncore_contract.FeatureTaskLabel("GKE Audit Logs", "GKE description", 100, true),
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"platform": "gke"}),
					coretask.WithSelectionPriority(100),
				),
			},
			wantAvailableTaskIDs: []string{"audit-log-parser#gke"},
			wantFeatureListIDs:   []string{"audit-log-parser#gke"},
			wantEnabledFeatures:  []string{"audit-log-parser#gke"},
		},
		{
			name: "higher priority task with defaultFeature false overrides generic defaultFeature true",
			inspectionType: InspectionType{
				Id:     "gke-inspection",
				Name:   "GKE Inspection",
				Labels: map[string]string{"platform": "gke"},
			},
			tasks: []coretask.UntypedTask{
				// Generic implementation with priority 0, default feature = true
				coretask.NewTask(
					taskid.NewImplementationID(taskid.NewTaskReference[any]("custom-feature"), "generic"),
					nil,
					func(ctx context.Context) (any, error) { return nil, nil },
					inspectioncore_contract.FeatureTaskLabel("Generic Feature", "Generic description", 100, true),
					coretask.WithSelectionPriority(0),
				),
				// GKE specialized implementation with priority 50, default feature = false
				coretask.NewTask(
					taskid.NewImplementationID(taskid.NewTaskReference[any]("custom-feature"), "gke"),
					nil,
					func(ctx context.Context) (any, error) { return nil, nil },
					inspectioncore_contract.FeatureTaskLabel("GKE Feature", "GKE description", 100, false),
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"platform": "gke"}),
					coretask.WithSelectionPriority(50),
				),
			},
			wantAvailableTaskIDs: []string{"custom-feature#gke"},
			wantFeatureListIDs:   []string{"custom-feature#gke"},
			wantEnabledFeatures:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore_contract.IOConfig{TemporaryFolder: t.TempDir()})
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}

			if err := server.AddInspectionType(tc.inspectionType); err != nil {
				t.Fatalf("AddInspectionType failed: %v", err)
			}

			for _, task := range tc.tasks {
				if err := server.AddTask(task); err != nil {
					t.Fatalf("AddTask failed: %v", err)
				}
			}

			inspectionID, err := server.CreateInspection(tc.inspectionType.Id)
			if err != nil {
				t.Fatalf("CreateInspection failed: %v", err)
			}
			runner := server.GetInspection(inspectionID)

			// Verify available tasks
			gotAvailableTasks := runner.availableTasks.GetAll()
			gotAvailableIDs := []string{}
			for _, task := range gotAvailableTasks {
				// Exclude internal serializer/inspection core tasks from assertion if any
				for _, expectedID := range tc.wantAvailableTaskIDs {
					if task.UntypedID().String() == expectedID {
						gotAvailableIDs = append(gotAvailableIDs, task.UntypedID().String())
					}
				}
			}
			if diff := cmp.Diff(tc.wantAvailableTaskIDs, gotAvailableIDs, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("availableTasks mismatch (-want +got):\n%s", diff)
			}

			// Verify FeatureList
			featureList, err := runner.FeatureList()
			if err != nil {
				t.Fatalf("FeatureList failed: %v", err)
			}
			gotFeatureListIDs := []string{}
			for _, f := range featureList {
				gotFeatureListIDs = append(gotFeatureListIDs, f.Id)
			}
			if diff := cmp.Diff(tc.wantFeatureListIDs, gotFeatureListIDs, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("FeatureList mismatch (-want +got):\n%s", diff)
			}

			// Verify enabledFeatures
			gotEnabledFeatures := []string{}
			for fID, enabled := range runner.enabledFeatures {
				if enabled {
					gotEnabledFeatures = append(gotEnabledFeatures, fID)
				}
			}
			if diff := cmp.Diff(tc.wantEnabledFeatures, gotEnabledFeatures, cmpopts.SortSlices(func(a, b string) bool { return a < b }), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("enabledFeatures mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInspectionTaskRunner_Cancel(t *testing.T) {
	logger.InitGlobalKHILogger()

	testCases := []struct {
		name       string
		failTask   bool
		setupRun   func(t *testing.T, runner *InspectionTaskRunner)
		wantErrSub string
	}{
		{
			name:       "cancel before start returns error",
			setupRun:   func(t *testing.T, runner *InspectionTaskRunner) {},
			wantErrSub: "this task is not yet started",
		},
		{
			name: "cancel after successful completion returns already finished error",
			setupRun: func(t *testing.T, runner *InspectionTaskRunner) {
				req := &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}
				if err := runner.Run(context.Background(), req); err != nil {
					t.Fatalf("Run failed: %v", err)
				}
				<-runner.Wait()
			},
			wantErrSub: "is already finished",
		},
		{
			name:     "cancel after failed completion returns already finished error",
			failTask: true,
			setupRun: func(t *testing.T, runner *InspectionTaskRunner) {
				req := &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}
				_ = runner.Run(context.Background(), req)
				<-runner.Wait()
			},
			wantErrSub: "is already finished",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore_contract.IOConfig{TemporaryFolder: t.TempDir()})
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}
			inspectionType := InspectionType{Id: "test-inspection", Name: "Test Inspection"}
			if err := server.AddInspectionType(inspectionType); err != nil {
				t.Fatalf("AddInspectionType failed: %v", err)
			}
			dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
			dummyTask := coretask.NewTask(
				dummyTaskID,
				nil,
				func(ctx context.Context) (any, error) {
					if tc.failTask {
						return nil, fmt.Errorf("simulated failure")
					}
					return "success", nil
				},
				coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
				coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
				coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
			)
			if err := server.AddTask(dummyTask); err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}
			inspectionID, err := server.CreateInspection(inspectionType.Id)
			if err != nil {
				t.Fatalf("CreateInspection failed: %v", err)
			}
			runner := server.GetInspection(inspectionID)

			tc.setupRun(t, runner)

			err = runner.Cancel()
			if err == nil {
				t.Fatalf("Cancel() expected error containing %q, got nil", tc.wantErrSub)
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("Cancel() error %q does not contain %q", err.Error(), tc.wantErrSub)
			}
		})
	}
}

func TestInspectionTaskRunner_TakeRunTaskGraphSnapshot(t *testing.T) {
	logger.InitGlobalKHILogger()

	testCases := []struct {
		name              string
		runInspection     bool
		failTask          bool
		wantErr           bool
		wantIsRunFinished bool
		wantPhase         coretask.TaskRunPhase
	}{
		{
			name:          "fails before the task graph execution starts",
			runInspection: false,
			wantErr:       true,
		},
		{
			name:              "reports done phase after a successful run",
			runInspection:     true,
			wantIsRunFinished: true,
			wantPhase:         coretask.TaskRunPhaseDone,
		},
		{
			name:              "reports error phase after a failed run",
			runInspection:     true,
			failTask:          true,
			wantIsRunFinished: true,
			wantPhase:         coretask.TaskRunPhaseError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var taskErr error
			if tc.failTask {
				taskErr = fmt.Errorf("simulated failure")
			}
			server, inspectionID, taskImplID := newTestInspectionServer(t, taskErr)
			runner := server.GetInspection(inspectionID)

			if tc.runInspection {
				req := &inspectioncore_contract.InspectionRequest{Values: map[string]any{}}
				if err := runner.Run(context.Background(), req); err != nil {
					t.Fatalf("Run failed: %v", err)
				}
				<-runner.Wait()
			}

			snapshot, err := runner.TakeRunTaskGraphSnapshot()
			if tc.wantErr {
				if !errors.Is(err, ErrRunTaskGraphNotStarted) {
					t.Fatalf("TakeRunTaskGraphSnapshot() error = %v, want ErrRunTaskGraphNotStarted", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("TakeRunTaskGraphSnapshot() failed: %v", err)
			}
			if snapshot.IsRunFinished != tc.wantIsRunFinished {
				t.Errorf("IsRunFinished = %v, want %v", snapshot.IsRunFinished, tc.wantIsRunFinished)
			}
			if !containsTaskImplID(snapshot.DAG, taskImplID) {
				t.Errorf("DAG does not contain %s", taskImplID)
			}
			status, found := snapshot.TaskRunStatuses[taskImplID]
			if !found {
				t.Fatalf("TaskRunStatuses has no entry for %s", taskImplID)
			}
			if status.Phase != tc.wantPhase {
				t.Errorf("TaskRunStatuses[%s].Phase = %v, want %v", taskImplID, status.Phase, tc.wantPhase)
			}
		})
	}
}

// containsTaskImplID reports whether the DAG holds a node with the given task implementation ID.
func containsTaskImplID(dag *apiv1.TaskDAGInfo, taskImplID string) bool {
	for _, node := range dag.GetNodes() {
		if node.GetTaskImplementationId() == taskImplID {
			return true
		}
	}
	return false
}

// newTestInspectionServer registers an inspection type holding a single feature task and creates one
// inspection for it. The registered task fails with taskErr when taskErr is not nil.
// It returns the server, the created inspection ID, and the implementation ID of the registered task.
func newTestInspectionServer(t *testing.T, taskErr error) (*InspectionTaskServer, string, string) {
	t.Helper()
	server, err := NewServer(&inspectioncore_contract.IOConfig{TemporaryFolder: t.TempDir()})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	inspectionType := InspectionType{Id: "test-inspection", Name: "Test Inspection"}
	if err := server.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("AddInspectionType failed: %v", err)
	}
	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	dummyTask := coretask.NewTask(
		dummyTaskID,
		nil,
		func(ctx context.Context) (any, error) {
			if taskErr != nil {
				return nil, taskErr
			}
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionFeatureFlag, true),
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
	)
	if err := server.AddTask(dummyTask); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	inspectionID, err := server.CreateInspection(inspectionType.Id)
	if err != nil {
		t.Fatalf("CreateInspection failed: %v", err)
	}
	return server, inspectionID, dummyTaskID.String()
}
