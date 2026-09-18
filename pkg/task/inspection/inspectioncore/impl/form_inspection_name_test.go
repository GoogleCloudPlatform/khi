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

func TestInputInspectionNameTask(t *testing.T) {
	testCases := []struct {
		name           string
		defaultName    string
		inputValue     *string
		taskMode       inspectioncore.InspectionTaskModeType
		wantResult     string
		wantHeaderName string
		wantHint       string
		wantHintType   inspectionmetadata.ParameterHintType
	}{
		{
			name:           "falls back to literal Inspection when context default name is unset",
			defaultName:    "",
			inputValue:     nil,
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "Inspection",
			wantHeaderName: "Inspection",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
		},
		{
			name:           "uses default name from context when no input is given",
			defaultName:    "Google Kubernetes Engine(1)",
			inputValue:     nil,
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "Google Kubernetes Engine(1)",
			wantHeaderName: "Google Kubernetes Engine(1)",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
		},
		{
			name:           "overrides default name with trimmed user input",
			defaultName:    "Google Kubernetes Engine",
			inputValue:     ptr("  My Production GKE Cluster  "),
			taskMode:       inspectioncore.TaskModeRun,
			wantResult:     "My Production GKE Cluster",
			wantHeaderName: "My Production GKE Cluster",
			wantHint:       "",
			wantHintType:   inspectionmetadata.None,
		},
		{
			name:           "rejects whitespace-only input with error hint and falls back to default",
			defaultName:    "Google Kubernetes Engine(2)",
			inputValue:     ptr("   "),
			taskMode:       inspectioncore.TaskModeDryRun,
			wantResult:     "Google Kubernetes Engine(2)",
			wantHeaderName: "Google Kubernetes Engine(2)",
			wantHint:       "inspection name must not be empty",
			wantHintType:   inspectionmetadata.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			if tc.defaultName != "" {
				ctx = khictx.WithValue(ctx, inspectioncore.DefaultInspectionName, tc.defaultName)
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
		})
	}
}

func ptr(s string) *string {
	return &s
}
