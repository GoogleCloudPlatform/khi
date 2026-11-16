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

package formtask

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestDefineCalloutForm_FormField(t *testing.T) {
	formID := taskid.NewDefaultImplementationID[struct{}]("cluster-prereq-callout")

	testCases := []struct {
		name           string
		spec           CalloutFormSpec
		wantFields     []inspectionmetadata.ParameterFormField
		wantDryRunErr  string
		wantRunModeErr bool
	}{
		{
			name: "registers info callout banner",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "Audit logs are queried from the default Cloud Logging bucket.", inspectionmetadata.Info, nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CalloutParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "cluster-prereq-callout",
						Priority:    10,
						Type:        inspectionmetadata.Callout,
						Label:       "Cluster Prerequisites",
						Description: "Environment prerequisite checks.",
						HintType:    inspectionmetadata.Info,
						Hint:        "Audit logs are queried from the default Cloud Logging bucket.",
					},
				},
			},
		},
		{
			name: "normalizes empty hintType to info when message is non-empty",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "Log Volume Advisory: narrow the query window for faster results.", "", nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CalloutParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "cluster-prereq-callout",
						Priority:    10,
						Type:        inspectionmetadata.Callout,
						Label:       "Cluster Prerequisites",
						Description: "Environment prerequisite checks.",
						HintType:    inspectionmetadata.Info,
						Hint:        "Log Volume Advisory: narrow the query window for faster results.",
					},
				},
			},
		},
		{
			name: "registers warning callout banner and succeeds in run mode",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "API Quota Warning: concurrent queries may be throttled.", inspectionmetadata.Warning, nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CalloutParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "cluster-prereq-callout",
						Priority:    10,
						Type:        inspectionmetadata.Callout,
						Label:       "Cluster Prerequisites",
						Description: "Environment prerequisite checks.",
						HintType:    inspectionmetadata.Warning,
						Hint:        "API Quota Warning: concurrent queries may be throttled.",
					},
				},
			},
		},
		{
			name: "registers error callout banner in dry run and fails in run mode",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "Missing Required Log Bucket: _Default bucket is disabled.", inspectionmetadata.Error, nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CalloutParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "cluster-prereq-callout",
						Priority:    10,
						Type:        inspectionmetadata.Callout,
						Label:       "Cluster Prerequisites",
						Description: "Environment prerequisite checks.",
						HintType:    inspectionmetadata.Error,
						Hint:        "Missing Required Log Bucket: _Default bucket is disabled.",
					},
				},
			},
			wantRunModeErr: true,
		},
		{
			name:       "omits field when Callout generator is nil",
			spec:       CalloutFormSpec{},
			wantFields: []inspectionmetadata.ParameterFormField{},
		},
		{
			name: "omits field when message is empty",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "", inspectionmetadata.Warning, nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{},
		},
		{
			name: "omits field when hintType is None",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "Suppressed notice", inspectionmetadata.None, nil
				},
			},
			wantFields: []inspectionmetadata.ParameterFormField{},
		},
		{
			name: "propagates unrecoverable error from Callout generator",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "", inspectionmetadata.Info, errors.New("backend check failed")
				},
			},
			wantDryRunErr: "backend check failed",
		},
		{
			name: "rejects unsupported hintType value",
			spec: CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					return "Invalid severity", inspectionmetadata.ParameterHintType("invalid"), nil
				},
			},
			wantDryRunErr: `unsupported hint type "invalid"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			taskDef := DefineCalloutForm(
				formID,
				10,
				"Cluster Prerequisites",
				"Environment prerequisite checks.",
				func(b *coretask.Binder) CalloutFormSpec {
					return tc.spec
				},
			)

			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, dryRunErr := inspectiontest.Run(t, dryRunCtx, taskDef, inspectioncore.TaskModeDryRun, map[string]any{})
			if tc.wantDryRunErr != "" {
				if dryRunErr == nil {
					t.Fatalf("dry run error = nil, want error containing %q", tc.wantDryRunErr)
				}
				if !strings.Contains(dryRunErr.Error(), tc.wantDryRunErr) {
					t.Fatalf("dry run error = %q, want error containing %q", dryRunErr.Error(), tc.wantDryRunErr)
				}
				return
			}
			if dryRunErr != nil {
				t.Fatalf("dry run unexpected error: %v", dryRunErr)
			}

			metadata := khictx.MustGetValue(dryRunCtx, inspectionmetadata.MapContextKey)
			fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatal("form field set metadata not found")
			}
			if diff := cmp.Diff(tc.wantFields, fields.Fields()); diff != "" {
				t.Errorf("Fields() mismatch (-want +got):\n%s", diff)
			}

			runCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, runMetadata, runErr := inspectiontest.Run(t, runCtx, taskDef, inspectioncore.TaskModeRun, map[string]any{})
			if tc.wantRunModeErr {
				if runErr == nil {
					t.Errorf("run mode error = nil, want validation error")
				}
			} else {
				if runErr != nil {
					t.Fatalf("run mode unexpected error: %v", runErr)
				}
				runFields, found := typedmap.Get(runMetadata, inspectionmetadata.FormFieldSetMetadataKey)
				if !found {
					t.Fatal("form field set metadata not found in run mode")
				}
				if diff := cmp.Diff(tc.wantFields, runFields.Fields()); diff != "" {
					t.Errorf("run mode Fields() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestDefineCalloutForm_BinderDependencies(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[int]("estimated-log-hours")
	formID := taskid.NewDefaultImplementationID[struct{}]("log-volume-advisory")

	task := DefineCalloutForm(
		formID,
		5,
		"Log Volume Advisory",
		"Checks query time window duration.",
		func(b *coretask.Binder) CalloutFormSpec {
			hours := coretask.Use(b, sourceTaskID.Ref())
			return CalloutFormSpec{
				Callout: func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error) {
					if hours.Get(ctx) > 12 {
						return "Wide time range selected; consider narrowing the window.", inspectionmetadata.Warning, nil
					}
					return "", inspectionmetadata.None, nil
				},
			}
		},
	)

	testCases := []struct {
		name       string
		hours      int
		wantFields []inspectionmetadata.ParameterFormField
	}{
		{
			name:  "emits warning callout when bound input exceeds threshold",
			hours: 24,
			wantFields: []inspectionmetadata.ParameterFormField{
				inspectionmetadata.CalloutParameterFormField{
					ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
						ID:          "log-volume-advisory",
						Priority:    5,
						Type:        inspectionmetadata.Callout,
						Label:       "Log Volume Advisory",
						Description: "Checks query time window duration.",
						HintType:    inspectionmetadata.Warning,
						Hint:        "Wide time range selected; consider narrowing the window.",
					},
				},
			},
		},
		{
			name:       "omits callout when bound input is within threshold",
			hours:      2,
			wantFields: []inspectionmetadata.ParameterFormField{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, metadata, err := inspectiontest.Run(
				t,
				ctx,
				task,
				inspectioncore.TaskModeDryRun,
				map[string]any{},
				tasktest.Given(sourceTaskID.Ref(), tc.hours),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatal("form field set not found on metadata")
			}
			if diff := cmp.Diff(tc.wantFields, fields.Fields()); diff != "" {
				t.Errorf("Fields() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("declares required input on source reference", func(t *testing.T) {
		wantInputs := []string{"required estimated-log-hours"}
		if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("has form task label", func(t *testing.T) {
		isFormTask, found := typedmap.Get(task.Labels(), inspectioncore.TaskLabelKeyIsFormTask)
		if !found || !isFormTask {
			t.Errorf("task label %v = %v, want true", inspectioncore.TaskLabelKeyIsFormTask, isFormTask)
		}
	})
}
