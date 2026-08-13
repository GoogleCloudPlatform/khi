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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
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
		coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionTypes, []string{inspectionType.Id}),
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
	runner := &InspectionTaskRunner{}

	tests := []struct {
		name        string
		taskLabels  map[string]any // simplified setup
		currentType *InspectionType
		want        bool
	}{
		{
			name: "Selector matches target labels",
			taskLabels: map[string]any{
				inspectioncore_contract.LabelKeyInspectionTypeLabelSelector.Key(): inspectioncore_contract.LabelSelector{"platform": "gke"},
			},
			currentType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gke", "provider": "google"},
			},
			want: true,
		},
		{
			name: "Selector does not match target labels",
			taskLabels: map[string]any{
				inspectioncore_contract.LabelKeyInspectionTypeLabelSelector.Key(): inspectioncore_contract.LabelSelector{"platform": "gke"},
			},
			currentType: &InspectionType{
				Id:     "some-env",
				Labels: map[string]string{"platform": "gdc"},
			},
			want: false,
		},
		{
			name: "Fallback to legacy list - match",
			taskLabels: map[string]any{
				inspectioncore_contract.LabelKeyInspectionTypes.Key(): []string{"legacy-env", "other-env"},
			},
			currentType: &InspectionType{
				Id: "legacy-env",
			},
			want: true,
		},
		{
			name: "Fallback to legacy list - no match",
			taskLabels: map[string]any{
				inspectioncore_contract.LabelKeyInspectionTypes.Key(): []string{"other-env"},
			},
			currentType: &InspectionType{
				Id: "legacy-env",
			},
			want: false,
		},
		{
			name:       "No selector, no legacy list (Global task)",
			taskLabels: map[string]any{},
			currentType: &InspectionType{
				Id: "any-env",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create task with labels
			opts := []coretask.LabelOpt{}
			for k, v := range tt.taskLabels {
				if k == inspectioncore_contract.LabelKeyInspectionTypeLabelSelector.Key() {
					opts = append(opts, inspectioncore_contract.InspectionTypeLabelSelector(v.(inspectioncore_contract.LabelSelector)))
				} else if k == inspectioncore_contract.LabelKeyInspectionTypes.Key() {
					opts = append(opts, coretask.WithLabelValue(inspectioncore_contract.LabelKeyInspectionTypes, v.([]string)))
				}
			}

			task := coretask.NewTask(
				taskid.NewDefaultImplementationID[any]("test-task"),
				nil,
				func(ctx context.Context) (any, error) { return nil, nil },
				opts...,
			)

			got := runner.isTaskCompatible(task, tt.currentType)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("isTaskCompatible() mismatch (-want +got):\n%s", diff)
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
