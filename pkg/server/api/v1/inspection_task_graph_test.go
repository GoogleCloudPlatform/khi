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
	"testing"

	"connectrpc.com/connect"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
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

	server := NewInspectionTaskGraphServer(inspectionServer)

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

	server := NewInspectionTaskGraphServer(inspectionServer)

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
