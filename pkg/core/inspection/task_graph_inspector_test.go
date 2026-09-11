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

package coreinspection

import (
	"context"
	"errors"
	"strings"
	"testing"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func createTestTask(refID, implHash string, deps []coretask.Dependency, opts ...coretask.LabelOpt) coretask.UntypedTask {
	ref := taskid.NewTaskReference[any](refID)
	id := taskid.NewImplementationID[any](ref, implHash)
	return coretask.NewTask[any](id, deps, func(ctx context.Context) (any, error) {
		return nil, nil
	}, opts...)
}

func TestEvaluateTaskCompatibility(t *testing.T) {
	inspectionType := &InspectionType{
		Id:   "gke-audit",
		Name: "GKE Audit",
		Labels: map[string]string{
			"environment": "googlecloud",
			"log_source":  "cloud_logging",
		},
	}

	testCases := []struct {
		name           string
		task           coretask.UntypedTask
		wantCompatible bool
		wantReasonSub  string
	}{
		{
			name: "matches label selector",
			task: createTestTask("task.a", "1", nil, inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
				"environment": "googlecloud",
				"log_source":  "cloud_logging",
			})),
			wantCompatible: true,
			wantReasonSub:  "Matched label selector",
		},
		{
			name: "fails label selector with mismatched value",
			task: createTestTask("task.a", "2", nil, inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
				"environment": "onprem",
			})),
			wantCompatible: false,
			wantReasonSub:  "Mismatched label selector",
		},
		{
			name: "fails label selector with missing key",
			task: createTestTask("task.a", "3", nil, inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{
				"cluster_type": "autopilot",
			})),
			wantCompatible: false,
			wantReasonSub:  "missing on inspection type",
		},
		{
			name:           "global task with no inspection constraints",
			task:           createTestTask("task.c", "1", nil),
			wantCompatible: true,
			wantReasonSub:  "Global task",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCompat, gotReason := EvaluateTaskCompatibility(tc.task, inspectionType)
			if gotCompat != tc.wantCompatible {
				t.Errorf("EvaluateTaskCompatibility() compatible = %v, want %v", gotCompat, tc.wantCompatible)
			}
			if !strings.Contains(gotReason, tc.wantReasonSub) {
				t.Errorf("EvaluateTaskCompatibility() reason = %q, want substring %q", gotReason, tc.wantReasonSub)
			}
		})
	}
}

func TestInspectRegistry(t *testing.T) {
	testCases := []struct {
		name                string
		setupServer         func(t *testing.T) *InspectionTaskServer
		wantInspectionTypes []*apiv1.RegisteredInspectionTypeInfo
		wantGroups          []*apiv1.RegisteredTaskGroupInfo
		wantErr             bool
	}{
		{
			name: "returns sorted task groups and inspection types",
			setupServer: func(t *testing.T) *InspectionTaskServer {
				server, err := NewServer(nil)
				if err != nil {
					t.Fatalf("failed to create server: %v", err)
				}
				err = server.AddInspectionType(InspectionType{
					Id:          "test-type",
					Name:        "Test Type",
					Description: "A test inspection type",
					Icon:        "test-icon",
					Labels: map[string]string{
						"env": "test",
					},
				})
				if err != nil {
					t.Fatalf("failed to add inspection type: %v", err)
				}

				depRef := taskid.NewTaskReference[any]("dep.ref")
				task1 := createTestTask("group.ref", "impl1", []coretask.Dependency{depRef},
					coretask.WithSelectionPriority(10),
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"env": "test"}),
				)
				task2 := createTestTask("group.ref", "impl2", nil,
					coretask.WithSelectionPriority(20),
					coretask.ProvidesTag(coretask.NewTag[string]("test.tag.b"), coretask.WithTagPriority(50)),
					coretask.ProvidesTag(coretask.NewTag[int]("test.tag.a"), coretask.WithTagPriority(10)),
				)
				task3 := createTestTask("dep.ref", "impl1", nil)

				_ = server.AddTask(task1)
				_ = server.AddTask(task2)
				_ = server.AddTask(task3)
				return server
			},
			wantInspectionTypes: []*apiv1.RegisteredInspectionTypeInfo{
				{
					Id:          proto.String("test-type"),
					Name:        proto.String("Test Type"),
					Description: proto.String("A test inspection type"),
					Icon:        proto.String("test-icon"),
					Labels: map[string]string{
						"env": "test",
					},
				},
			},
			wantGroups: []*apiv1.RegisteredTaskGroupInfo{
				{
					TaskReferenceId: proto.String("dep.ref"),
					Tasks: []*apiv1.RegisteredTaskInfo{
						{
							TaskImplementationId: proto.String("dep.ref#impl1"),
							TaskReferenceId:      proto.String("dep.ref"),
							Priority:             proto.Int32(0),
							IsFeature:            proto.Bool(false),
							IsDefaultFeature:     proto.Bool(false),
							FeatureLabel:         proto.String(""),
							FeatureDescription:   proto.String(""),
							Dependencies:         []*apiv1.TaskDependencyInfo{},
							Labels: map[string]string{
								coretask.LabelKeyTaskResultType.Key(): "interface {}",
							},
							OutputType: proto.String("interface {}"),
						},
					},
				},
				{
					TaskReferenceId: proto.String("group.ref"),
					Tasks: []*apiv1.RegisteredTaskInfo{
						{
							TaskImplementationId: proto.String("group.ref#impl2"),
							TaskReferenceId:      proto.String("group.ref"),
							Priority:             proto.Int32(20),
							IsFeature:            proto.Bool(false),
							IsDefaultFeature:     proto.Bool(false),
							FeatureLabel:         proto.String(""),
							FeatureDescription:   proto.String(""),
							ProvidedTags: []*apiv1.ProvidedTagInfo{
								{
									Tag:        proto.String("test.tag.a"),
									Priority:   proto.Int32(10),
									OutputType: proto.String("int"),
								},
								{
									Tag:        proto.String("test.tag.b"),
									Priority:   proto.Int32(50),
									OutputType: proto.String("string"),
								},
							},
							Dependencies:         []*apiv1.TaskDependencyInfo{},
							SelectorRequirements: []*apiv1.LabelSelectorRequirementInfo{},
							Labels: map[string]string{
								coretask.LabelKeyTaskSelectionPriority.Key():             "20",
								coretask.LabelKeyTaskResultType.Key():                    "interface {}",
								coretask.LabelKeyProvidedTag("test.tag.a").Key():         "true",
								coretask.LabelKeyProvidedTagPriority("test.tag.a").Key(): "10",
								coretask.LabelKeyProvidedTagType("test.tag.a").Key():     "int",
								coretask.LabelKeyProvidedTag("test.tag.b").Key():         "true",
								coretask.LabelKeyProvidedTagPriority("test.tag.b").Key(): "50",
								coretask.LabelKeyProvidedTagType("test.tag.b").Key():     "string",
							},
							OutputType: proto.String("interface {}"),
						},
						{
							TaskImplementationId: proto.String("group.ref#impl1"),
							TaskReferenceId:      proto.String("group.ref"),
							Priority:             proto.Int32(10),
							IsFeature:            proto.Bool(false),
							IsDefaultFeature:     proto.Bool(false),
							FeatureLabel:         proto.String(""),
							FeatureDescription:   proto.String(""),
							Dependencies: []*apiv1.TaskDependencyInfo{
								{
									Cardinality:       apiv1.TaskDependencyCardinality_TASK_DEPENDENCY_CARDINALITY_POINT_TO_POINT.Enum(),
									Scope:             apiv1.TaskDependencyScope_TASK_DEPENDENCY_SCOPE_ALL.Enum(),
									TargetReferenceId: proto.String("dep.ref"),
									OutputType:        proto.String("interface {}"),
								},
							},
							SelectorRequirements: []*apiv1.LabelSelectorRequirementInfo{
								{
									Key:      proto.String("env"),
									Operator: proto.String("=="),
									Values:   []string{"test"},
								},
							},
							Labels: map[string]string{
								inspectioncore_contract.LabelKeyInspectionTypeLabelSelector.Key(): "map[env:test]",
								coretask.LabelKeyTaskSelectionPriority.Key():                      "10",
								coretask.LabelKeyTaskResultType.Key():                             "interface {}",
							},
							OutputType: proto.String("interface {}"),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer(t)
			got, err := InspectRegistry(server)
			if (err != nil) != tc.wantErr {
				t.Fatalf("InspectRegistry() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantInspectionTypes, got.GetInspectionTypes(), protocmp.Transform()); diff != "" {
				t.Errorf("InspectRegistry() InspectionTypes mismatch (-want +got):\n%s", diff)
			}
			var filteredGroups []*apiv1.RegisteredTaskGroupInfo
			for _, g := range got.GetTaskGroups() {
				if g.GetTaskReferenceId() == "dep.ref" || g.GetTaskReferenceId() == "group.ref" {
					filteredGroups = append(filteredGroups, g)
				}
			}
			if diff := cmp.Diff(tc.wantGroups, filteredGroups, protocmp.Transform()); diff != "" {
				t.Errorf("InspectRegistry() TaskGroups mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInspectResolution(t *testing.T) {
	testCases := []struct {
		name              string
		setupServer       func(t *testing.T) *InspectionTaskServer
		inspectionTypeID  string
		featureOverrides  map[string]bool
		wantErr           bool
		wantSuccess       bool
		wantEvaluationLen int
		checkResponse     func(t *testing.T, resp *apiv1.ResolveInspectionTaskGraphResponse)
	}{
		{
			name: "unknown inspection type returns error",
			setupServer: func(t *testing.T) *InspectionTaskServer {
				server, _ := NewServer(nil)
				return server
			},
			inspectionTypeID: "nonexistent",
			wantErr:          true,
		},
		{
			name: "successful resolution with priority competition and features",
			setupServer: func(t *testing.T) *InspectionTaskServer {
				server, _ := NewServer(nil)
				_ = server.AddInspectionType(InspectionType{
					Id:     "cluster-audit",
					Labels: map[string]string{"env": "gke"},
				})

				taskA1 := createTestTask("feature.a", "impl1", nil,
					coretask.WithSelectionPriority(10),
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"env": "gke"}),
					inspectioncore_contract.FeatureTaskLabel("Feature A", "Description A", 1, true),
				)
				taskA2 := createTestTask("feature.a", "impl2", nil,
					coretask.WithSelectionPriority(20),
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"env": "gke"}),
					inspectioncore_contract.FeatureTaskLabel("Feature A", "Description A", 1, true),
				)
				taskB := createTestTask("task.b", "impl1", nil,
					inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"env": "onprem"}),
				)
				refA := taskid.NewTaskReference[any]("feature.a")
				taskC := createTestTask("task.c", "impl1", []coretask.Dependency{refA},
					inspectioncore_contract.FeatureTaskLabel("Feature C", "Description C", 2, true),
				)

				_ = server.AddTask(taskA1)
				_ = server.AddTask(taskA2)
				_ = server.AddTask(taskB)
				_ = server.AddTask(taskC)
				return server
			},
			inspectionTypeID: "cluster-audit",
			featureOverrides: map[string]bool{"feature.a#impl2": true},
			wantErr:          false,
			wantSuccess:      true,
			checkResponse: func(t *testing.T, resp *apiv1.ResolveInspectionTaskGraphResponse) {
				evalMap := make(map[string]*apiv1.TaskFilterEvaluation)
				for _, ev := range resp.GetFilteringEvaluations() {
					evalMap[ev.GetTaskImplementationId()] = ev
				}

				evA2 := evalMap["feature.a#impl2"]
				if evA2 == nil || !evA2.GetIsCompatible() || !evA2.GetIsSelected() {
					t.Errorf("evA2 expected compatible and selected, got %+v", evA2)
				}

				evA1 := evalMap["feature.a#impl1"]
				if evA1 == nil || !evA1.GetIsCompatible() || evA1.GetIsSelected() || evA1.GetSupersededByTaskImplementationId() != "feature.a#impl2" {
					t.Errorf("evA1 expected compatible but superseded by feature.a#impl2, got %+v", evA1)
				}

				evB := evalMap["task.b#impl1"]
				if evB == nil || evB.GetIsCompatible() {
					t.Errorf("evB expected incompatible, got %+v", evB)
				}

				dag := resp.GetDag()
				if !dag.GetIsSuccess() {
					t.Fatalf("DAG expected success, got error: %s", dag.GetErrorMessage())
				}

				nodeIDs := make(map[string]bool)
				for _, n := range dag.GetNodes() {
					nodeIDs[n.GetTaskImplementationId()] = true
				}
				if !nodeIDs["feature.a#impl2"] {
					t.Errorf("DAG nodes missing feature.a#impl2")
				}
				if !nodeIDs["task.c#impl1"] {
					t.Errorf("DAG nodes missing task.c#impl1")
				}
				if nodeIDs["task.b#impl1"] {
					t.Errorf("DAG nodes should not contain incompatible task.b#impl1")
				}

				foundEdge := false
				for _, e := range dag.GetEdges() {
					if e.GetSourceImplementationId() == "feature.a#impl2" && e.GetDestinationImplementationId() == "task.c#impl1" {
						foundEdge = true
						break
					}
				}
				if !foundEdge {
					t.Errorf("DAG edges missing feature.a#impl2 -> task.c#impl1")
				}
			},
		},
		{
			name: "resolution failure due to cycle returns is_success=false and diagnostic message without HTTP error",
			setupServer: func(t *testing.T) *InspectionTaskServer {
				server, _ := NewServer(nil)
				_ = server.AddInspectionType(InspectionType{
					Id: "cycle-type",
				})

				refY := taskid.NewTaskReference[any]("task.y")
				refX := taskid.NewTaskReference[any]("task.x")

				taskX := createTestTask("task.x", "impl", []coretask.Dependency{refY},
					inspectioncore_contract.FeatureTaskLabel("Feature X", "Desc", 1, true),
				)
				taskY := createTestTask("task.y", "impl", []coretask.Dependency{refX})

				_ = server.AddTask(taskX)
				_ = server.AddTask(taskY)
				return server
			},
			inspectionTypeID: "cycle-type",
			featureOverrides: map[string]bool{"task.x#impl": true},
			wantErr:          false,
			wantSuccess:      false,
			checkResponse: func(t *testing.T, resp *apiv1.ResolveInspectionTaskGraphResponse) {
				dag := resp.GetDag()
				if dag.GetIsSuccess() {
					t.Errorf("DAG expected failure due to cycle, but succeeded")
				}
				if dag.GetErrorMessage() == "" {
					t.Errorf("DAG expected non-empty error message on failure")
				}
				if len(resp.GetFilteringEvaluations()) == 0 {
					t.Errorf("Filtering evaluations should be populated even when DAG resolution fails")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer(t)
			resp, err := InspectResolution(server, tc.inspectionTypeID, tc.featureOverrides)
			if (err != nil) != tc.wantErr {
				t.Fatalf("InspectResolution() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if !errors.Is(err, ErrInspectionTypeNotFound) {
					t.Errorf("expected ErrInspectionTypeNotFound, got %v", err)
				}
				return
			}

			if resp.GetDag().GetIsSuccess() != tc.wantSuccess {
				t.Errorf("resp.Dag.IsSuccess = %v, want %v (error: %s)", resp.GetDag().GetIsSuccess(), tc.wantSuccess, resp.GetDag().GetErrorMessage())
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, resp)
			}
		})
	}
}
