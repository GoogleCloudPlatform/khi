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

package inspectioncore_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TestInputInspectionNameTask verifies default name resolution, user overrides, name collision validation, and dry-run fallback.
func TestInputInspectionNameTask(t *testing.T) {
	testCases := []struct {
		name                 string
		typeName             string
		inspectionID         string
		setupRegistry        func(r inspectioncore.InspectionNameRegistry)
		inputValue           *string
		taskMode             inspectioncore.InspectionTaskModeType
		wantResult           string
		wantHeaderName       string
		wantHint             string
		wantHintType         inspectionmetadata.ParameterHintType
		verifyAfterExecution func(t *testing.T, r inspectioncore.InspectionNameRegistry)
	}{
		{
			name:           "reserves default name on initial run",
			typeName:       "Google Kubernetes Engine",
			inspectionID:   "insp-1",
			inputValue:     nil,
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "Google Kubernetes Engine",
			wantHeaderName: "Google Kubernetes Engine",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
			verifyAfterExecution: func(t *testing.T, r inspectioncore.InspectionNameRegistry) {
				if got := r.ResolveUniqueName("insp-2", "Google Kubernetes Engine"); got != "Google Kubernetes Engine(1)" {
					t.Errorf("ResolveUniqueName() for insp-2 = %q, want %q", got, "Google Kubernetes Engine(1)")
				}
			},
		},
		{
			name:         "second inspection automatically gets suffix 1 when base name is reserved",
			typeName:     "Google Kubernetes Engine",
			inspectionID: "insp-2",
			setupRegistry: func(r inspectioncore.InspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Google Kubernetes Engine")
			},
			inputValue:     nil,
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "Google Kubernetes Engine(1)",
			wantHeaderName: "Google Kubernetes Engine(1)",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
		},
		{
			name:           "overrides default name with trimmed user input and reserves it",
			typeName:       "Google Kubernetes Engine",
			inspectionID:   "insp-1",
			inputValue:     ptr("  My Production Cluster  "),
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "My Production Cluster",
			wantHeaderName: "My Production Cluster",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
			verifyAfterExecution: func(t *testing.T, r inspectioncore.InspectionNameRegistry) {
				if err := r.ReserveName("insp-2", "My Production Cluster"); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected ErrInspectionNameAlreadyInUse for My Production Cluster, got %v", err)
				}
			},
		},
		{
			name:         "entering a name already reserved by another inspection produces error hint and falls back to default",
			typeName:     "Google Kubernetes Engine",
			inspectionID: "insp-2",
			setupRegistry: func(r inspectioncore.InspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Taken Name")
			},
			inputValue:     ptr("Taken Name"),
			taskMode:       inspectioncore.TaskModeDryRun,
			wantResult:     "Google Kubernetes Engine",
			wantHeaderName: "Google Kubernetes Engine",
			wantHint:       "inspection name \"Taken Name\" is already in use",
			wantHintType:   inspectionmetadata.Error,
			verifyAfterExecution: func(t *testing.T, r inspectioncore.InspectionNameRegistry) {
				if err := r.ReserveName("insp-3", "Taken Name"); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected Taken Name to remain reserved by insp-1, got %v", err)
				}
			},
		},
		{
			name:         "entering duplicate name in DryRun preserves previous custom reservation",
			typeName:     "Google Kubernetes Engine",
			inspectionID: "insp-1",
			setupRegistry: func(r inspectioncore.InspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Previous Custom Name")
				_ = r.ReserveName("insp-2", "Taken Name")
			},
			inputValue:     ptr("Taken Name"),
			taskMode:       inspectioncore.TaskModeDryRun,
			wantResult:     "Google Kubernetes Engine",
			wantHeaderName: "Google Kubernetes Engine",
			wantHint:       "inspection name \"Taken Name\" is already in use",
			wantHintType:   inspectionmetadata.Error,
			verifyAfterExecution: func(t *testing.T, r inspectioncore.InspectionNameRegistry) {
				if err := r.ReserveName("other-insp", "Previous Custom Name"); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected Previous Custom Name to remain reserved for insp-1, got %v", err)
				}
				if err := r.ReserveName("other-insp", "Taken Name"); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected Taken Name to remain reserved for insp-2, got %v", err)
				}
			},
		},
		{
			name:         "entering empty name in DryRun preserves previous custom reservation",
			typeName:     "Google Kubernetes Engine",
			inspectionID: "insp-1",
			setupRegistry: func(r inspectioncore.InspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Previous Custom Name")
			},
			inputValue:     ptr(""),
			taskMode:       inspectioncore.TaskModeDryRun,
			wantResult:     "Google Kubernetes Engine",
			wantHeaderName: "Google Kubernetes Engine",
			wantHint:       "inspection name must not be empty",
			wantHintType:   inspectionmetadata.Error,
			verifyAfterExecution: func(t *testing.T, r inspectioncore.InspectionNameRegistry) {
				if err := r.ReserveName("other-insp", "Previous Custom Name"); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected Previous Custom Name to remain reserved for insp-1, got %v", err)
				}
			},
		},
		{
			name:           "rejects whitespace-only input with error hint and falls back to default",
			typeName:       "Google Kubernetes Engine",
			inspectionID:   "insp-1",
			inputValue:     ptr("   "),
			taskMode:       inspectioncore.TaskModeDryRun,
			wantResult:     "Google Kubernetes Engine",
			wantHeaderName: "Google Kubernetes Engine",
			wantHint:       "inspection name must not be empty",
			wantHintType:   inspectionmetadata.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			var registry inspectioncore.InspectionNameRegistry = inspectioncore.NewInMemoryInspectionNameRegistry()
			if tc.setupRegistry != nil {
				tc.setupRegistry(registry)
			}
			ctx = khictx.WithValue(ctx, inspectioncore.InspectionNameRegistryKey, registry)
			ctx = khictx.WithValue(ctx, inspectioncore.InspectionTypeName, tc.typeName)
			if tc.inspectionID != "" {
				ctx = khictx.WithValue(ctx, inspectioncore.InspectionTaskInspectionID, tc.inspectionID)
			}

			taskInput := map[string]any{}
			if tc.inputValue != nil {
				taskInput[inspectioncore.InputInspectionNameTaskID.ReferenceIDString()] = *tc.inputValue
			}

			got, _, err := inspectiontest.RunInspectionTask(ctx, InputInspectionNameTask, tc.taskMode, taskInput)
			if err != nil {
				t.Fatalf("RunInspectionTask() failed: %v", err)
			}

			if got != tc.wantResult {
				t.Errorf("InputInspectionNameTask result = %q, want %q", got, tc.wantResult)
			}

			metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey)
			if !found || header == nil {
				t.Fatalf("header metadata was not found in metadata set")
			}
			if header.InspectionName != tc.wantHeaderName {
				t.Errorf("header.InspectionName = %q, want %q", header.InspectionName, tc.wantHeaderName)
			}
			wantSuggestedFileName := tc.wantHeaderName + ".khi"
			if header.SuggestedFileName != wantSuggestedFileName {
				t.Errorf("header.SuggestedFileName = %q, want %q", header.SuggestedFileName, wantSuggestedFileName)
			}

			formFields, found := typedmap.Get(metadataSet, inspectionmetadata.FormFieldSetMetadataKey)
			if !found || formFields == nil {
				t.Fatalf("form field metadata was not found in metadata set")
			}
			rawField := formFields.DangerouslyGetField(inspectioncore.InputInspectionNameTaskID.ReferenceIDString())
			textField, ok := rawField.(inspectionmetadata.TextParameterFormField)
			if !ok {
				t.Fatalf("expected TextParameterFormField, got %T", rawField)
			}
			if textField.Hint != tc.wantHint {
				t.Errorf("textField.Hint = %q, want %q", textField.Hint, tc.wantHint)
			}
			if textField.HintType != tc.wantHintType {
				t.Errorf("textField.HintType = %q, want %q", textField.HintType, tc.wantHintType)
			}
			if textField.Priority != inspectionNameFormPriority {
				t.Errorf("textField.Priority = %d, want %d", textField.Priority, inspectionNameFormPriority)
			}
			if textField.Type != inspectionmetadata.Text {
				t.Errorf("textField.Type = %q, want %q", textField.Type, inspectionmetadata.Text)
			}

			if tc.verifyAfterExecution != nil {
				tc.verifyAfterExecution(t, registry)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}
