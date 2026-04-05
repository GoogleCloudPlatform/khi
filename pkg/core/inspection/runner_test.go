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
