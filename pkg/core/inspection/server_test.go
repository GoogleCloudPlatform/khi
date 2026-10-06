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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

// TestRegisterImportedInspection_NameReservation verifies auto-numbering and reservation when importing inspections.
func TestRegisterImportedInspection_NameReservation(t *testing.T) {
	testCases := []struct {
		name                string
		importInspections   []string
		wantInspectionNames []string
		wantSuggestedFiles  []string
	}{
		{
			name:                "sequential imported inspections with the same name get auto-numbered and reserved",
			importInspections:   []string{"Google Kubernetes Engine", "Google Kubernetes Engine", "Google Kubernetes Engine"},
			wantInspectionNames: []string{"Google Kubernetes Engine", "Google Kubernetes Engine(1)", "Google Kubernetes Engine(2)"},
			wantSuggestedFiles:  []string{"Google Kubernetes Engine.khi", "Google Kubernetes Engine(1).khi", "Google Kubernetes Engine(2).khi"},
		},
		{
			name:                "empty inspection name falls back to Inspection and auto-numbers",
			importInspections:   []string{"", "   "},
			wantInspectionNames: []string{"Inspection", "Inspection(1)"},
			wantSuggestedFiles:  []string{"Inspection.khi", "Inspection(1).khi"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore.IOConfig{})
			if err != nil {
				t.Fatalf("failed to create inspection task server: %v", err)
			}

			for i, baseName := range tc.importInspections {
				id := fmt.Sprintf("imported-%d", i)
				md := typedmap.NewTypedMap()
				header := &inspectionmetadata.HeaderMetadata{
					InspectionName: baseName,
				}
				typedmap.Set(md, inspectionmetadata.HeaderMetadataKey, header)

				runner := server.RegisterImportedInspection(id, nil, md.AsReadonly())
				if runner == nil {
					t.Fatalf("RegisterImportedInspection returned nil")
				}

				wantName := tc.wantInspectionNames[i]
				if header.InspectionName != wantName {
					t.Errorf("imported inspection [%d] InspectionName = %q, want %q", i, header.InspectionName, wantName)
				}
				wantFile := tc.wantSuggestedFiles[i]
				if header.SuggestedFileName != wantFile {
					t.Errorf("imported inspection [%d] SuggestedFileName = %q, want %q", i, header.SuggestedFileName, wantFile)
				}

				if err := server.InspectionNameRegistry().ReserveName("other-id", wantName); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected %q to be reserved, but got error: %v", wantName, err)
				}
			}
		})
	}
}

func TestInspectionTaskServer_AddModules(t *testing.T) {
	testCases := []struct {
		name                  string
		modules               []Module
		wantInspectionTypeIDs []string
		wantSelectors         map[string]inspectioncore.LabelSelector
		wantErrSubstr         string
	}{
		{
			name: "registers inspection types and scoped tasks of all modules",
			modules: []Module{
				{
					Name:            "first",
					Scope:           Scope{"env": "cloud"},
					InspectionTypes: []InspectionType{{Id: "type-a"}},
					Tasks:           []coretask.UntypedTask{newModuleTestTask("task-a")},
				},
				{
					Name:  "second",
					Tasks: []coretask.UntypedTask{newModuleTestTask("task-b")},
				},
			},
			wantInspectionTypeIDs: []string{"type-a"},
			wantSelectors: map[string]inspectioncore.LabelSelector{
				"task-a": {"env": "cloud"},
				"task-b": nil,
			},
		},
		{
			name: "duplicated task ID returns error with module name",
			modules: []Module{
				{Name: "first", Tasks: []coretask.UntypedTask{newModuleTestTask("task-a")}},
				{Name: "second", Tasks: []coretask.UntypedTask{newModuleTestTask("task-a")}},
			},
			wantErrSubstr: "module second: task id:task-a",
		},
		{
			name: "duplicated inspection type ID returns error with module name",
			modules: []Module{
				{Name: "first", InspectionTypes: []InspectionType{{Id: "type-a"}}},
				{Name: "second", InspectionTypes: []InspectionType{{Id: "type-a"}}},
			},
			wantErrSubstr: "module second: inspection type id:type-a is duplicated",
		},
		{
			name: "scope conflict returns error",
			modules: []Module{{
				Name:       "parent",
				Scope:      Scope{"cluster": "gke"},
				SubModules: []Module{{Name: "child", Scope: Scope{"cluster": "gdcv"}}},
			}},
			wantErrSubstr: "module parent/child: scope condition",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore.IOConfig{})
			if err != nil {
				t.Fatalf("failed to create inspection task server: %v", err)
			}

			err = server.AddModules(tc.modules...)
			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("AddModules() returned nil error, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("AddModules() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AddModules() returned unexpected error: %v", err)
			}
			for _, id := range tc.wantInspectionTypeIDs {
				if server.GetInspectionType(id) == nil {
					t.Errorf("GetInspectionType(%q) = nil, want the registered inspection type", id)
				}
			}
			var moduleTasks []coretask.UntypedTask
			for _, task := range server.GetAllRegisteredTasks() {
				if _, found := tc.wantSelectors[task.UntypedID().ReferenceIDString()]; found {
					moduleTasks = append(moduleTasks, task)
				}
			}
			if diff := cmp.Diff(tc.wantSelectors, selectorsByReferenceID(moduleTasks)); diff != "" {
				t.Errorf("registered task selectors mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
