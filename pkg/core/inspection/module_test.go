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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func newModuleTestTask(id string) coretask.UntypedTask {
	return coretask.DefineConstant(taskid.NewDefaultImplementationID[struct{}](id), struct{}{})
}

// selectorsByReferenceID returns the inspection type selector of each task keyed by its reference ID. Tasks without a selector map to nil.
func selectorsByReferenceID(tasks []coretask.UntypedTask) map[string]inspectioncore.LabelSelector {
	selectors := map[string]inspectioncore.LabelSelector{}
	for _, task := range tasks {
		selector, _ := typedmap.Get(task.Labels(), inspectioncore.LabelKeyInspectionTypeLabelSelector)
		selectors[task.UntypedID().ReferenceIDString()] = selector
	}
	return selectors
}

func TestModuleFlatten(t *testing.T) {
	testCases := []struct {
		name                  string
		module                Module
		wantSelectors         map[string]inspectioncore.LabelSelector
		wantInspectionTypeIDs []string
		wantErrSubstr         string
	}{
		{
			name: "module scope is applied to its tasks",
			module: Module{
				Name:  "parent",
				Scope: Scope{"env": "cloud"},
				Tasks: []coretask.UntypedTask{newModuleTestTask("task-a")},
			},
			wantSelectors: map[string]inspectioncore.LabelSelector{
				"task-a": {"env": "cloud"},
			},
		},
		{
			name: "tasks in an empty scope have no selector",
			module: Module{
				Name:  "parent",
				Tasks: []coretask.UntypedTask{newModuleTestTask("task-a")},
			},
			wantSelectors: map[string]inspectioncore.LabelSelector{
				"task-a": nil,
			},
		},
		{
			name: "sub-module scope adds conditions to the parent scope",
			module: Module{
				Name:  "parent",
				Scope: Scope{"env": "cloud"},
				Tasks: []coretask.UntypedTask{newModuleTestTask("task-parent")},
				SubModules: []Module{{
					Name:  "child",
					Scope: Scope{"source": "logging"},
					Tasks: []coretask.UntypedTask{newModuleTestTask("task-child")},
					SubModules: []Module{{
						Name:  "grandchild",
						Scope: Scope{"cluster": "gke"},
						Tasks: []coretask.UntypedTask{newModuleTestTask("task-grandchild")},
					}},
				}},
			},
			wantSelectors: map[string]inspectioncore.LabelSelector{
				"task-parent":     {"env": "cloud"},
				"task-child":      {"env": "cloud", "source": "logging"},
				"task-grandchild": {"env": "cloud", "source": "logging", "cluster": "gke"},
			},
		},
		{
			name: "sub-module can repeat a parent condition with the same value",
			module: Module{
				Name:  "parent",
				Scope: Scope{"env": "cloud"},
				SubModules: []Module{{
					Name:  "child",
					Scope: Scope{"env": "cloud", "source": "logging"},
					Tasks: []coretask.UntypedTask{newModuleTestTask("task-child")},
				}},
			},
			wantSelectors: map[string]inspectioncore.LabelSelector{
				"task-child": {"env": "cloud", "source": "logging"},
			},
		},
		{
			name: "inspection types are collected from sub-modules",
			module: Module{
				Name:            "parent",
				Scope:           Scope{"env": "cloud"},
				InspectionTypes: []InspectionType{{Id: "type-parent"}},
				SubModules: []Module{{
					Name:            "child",
					InspectionTypes: []InspectionType{{Id: "type-child"}},
				}},
			},
			wantSelectors:         map[string]inspectioncore.LabelSelector{},
			wantInspectionTypeIDs: []string{"type-parent", "type-child"},
		},
		{
			name: "sub-module scope conflicting with the parent scope returns error",
			module: Module{
				Name:  "parent",
				Scope: Scope{"cluster": "gke"},
				SubModules: []Module{{
					Name: "child",
					SubModules: []Module{{
						Name:  "grandchild",
						Scope: Scope{"cluster": "gdcv"},
					}},
				}},
			},
			wantErrSubstr: `module parent/child/grandchild: scope condition cluster="gdcv" conflicts with cluster="gke" in the parent scope`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.module.flatten()
			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("flatten() returned nil error, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("flatten() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("flatten() returned unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantSelectors, selectorsByReferenceID(got.tasks)); diff != "" {
				t.Errorf("flatten() task selectors mismatch (-want +got):\n%s", diff)
			}
			var gotInspectionTypeIDs []string
			for _, inspectionType := range got.inspectionTypes {
				gotInspectionTypeIDs = append(gotInspectionTypeIDs, inspectionType.Id)
			}
			if diff := cmp.Diff(tc.wantInspectionTypeIDs, gotInspectionTypeIDs); diff != "" {
				t.Errorf("flatten() inspection type IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
