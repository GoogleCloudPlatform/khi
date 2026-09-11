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

package googlecloudk8scommon_impl

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var expectedLegacyDefaultKinds = func() []string {
	kinds := strings.Split("pods replicasets daemonsets nodes deployments namespaces statefulsets services servicenetworkendpointgroups ingresses poddisruptionbudgets jobs cronjobs endpointslices persistentvolumes persistentvolumeclaims storageclasses horizontalpodautoscalers verticalpodautoscalers multidimpodautoscalers", " ")
	slices.Sort(kinds)
	return kinds
}()

func TestInputKindFilterTask_Metadata(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
	_, metadata, err := inspectiontest.RunInspectionTask(ctx, InputKindFilterTask, inspectioncore_contract.TaskModeDryRun, nil)
	if err != nil {
		t.Fatalf("unexpected error on DryRun mode: %v", err)
	}

	fields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
	if !found {
		t.Fatal("FormFieldSet not found on metadata")
	}

	rawField := fields.DangerouslyGetField(googlecloudk8scommon_contract.InputKindFilterTaskID.ReferenceIDString())
	field, ok := rawField.(inspectionmetadata.SetParameterFormField)
	if !ok {
		t.Fatalf("expected SetParameterFormField, got %T", rawField)
	}

	wantDefault := []string{"@any", "-leases"}
	if diff := cmp.Diff(wantDefault, field.Default); diff != "" {
		t.Errorf("default value mismatch (-want +got):\n%s", diff)
	}

	wantOptions := []inspectionmetadata.SetParameterFormFieldOptionItem{
		{ID: "@any", Description: "[Alias] An alias matches any of the kinds"},
		{ID: "@legacy_default", Description: "[Alias] An alias matches a set of kinds frequently queried in legacy KHI versions."},
	}
	if diff := cmp.Diff(wantOptions, field.Options); diff != "" {
		t.Errorf("options mismatch (-want +got):\n%s", diff)
	}
}

func TestInputKindFilterTask_Run(t *testing.T) {
	testCases := []struct {
		name       string
		inputValue any
		wantResult *gcpqueryutil.SetFilterParseResult
		wantErrSub string
	}{
		{
			name:       "default value used when input is nil",
			inputValue: nil,
			wantResult: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"leases"},
				Additives:    []string{},
			},
		},
		{
			name:       "explicit default @any -leases",
			inputValue: []any{"@any", "-leases"},
			wantResult: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"leases"},
				Additives:    []string{},
			},
		},
		{
			name:       "legacy_default alias expands to legacy kinds",
			inputValue: []any{"@legacy_default"},
			wantResult: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Subtractives: []string{},
				Additives:    expectedLegacyDefaultKinds,
			},
		},
		{
			name:       "legacy_default with subtractive element",
			inputValue: []any{"@legacy_default", "-pods"},
			wantResult: func() *gcpqueryutil.SetFilterParseResult {
				withoutPods := make([]string, 0, len(expectedLegacyDefaultKinds)-1)
				for _, k := range expectedLegacyDefaultKinds {
					if k != "pods" {
						withoutPods = append(withoutPods, k)
					}
				}
				return &gcpqueryutil.SetFilterParseResult{
					SubtractMode: false,
					Subtractives: []string{},
					Additives:    withoutPods,
				}
			}(),
		},
		{
			name:       "@any with multiple subtractive kinds",
			inputValue: []any{"@any", "-leases", "-configmaps"},
			wantResult: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"configmaps", "leases"},
				Additives:    []string{},
			},
		},
		{
			name:       "custom kinds",
			inputValue: []any{"pods", "services"},
			wantResult: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Subtractives: []string{},
				Additives:    []string{"pods", "services"},
			},
		},
		{
			name:       "empty filter produces validation error",
			inputValue: []any{},
			wantErrSub: "kind filter can't be empty",
		},
		{
			name:       "old @default alias is rejected",
			inputValue: []any{"@default"},
			wantErrSub: "alias `default` was not found",
		},
		{
			name:       "invalid character produces validation error",
			inputValue: []any{"invalid$$$"},
			wantErrSub: "filter value must be whitespace split series",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			inputMap := map[string]any{}
			if tc.inputValue != nil {
				inputMap[googlecloudk8scommon_contract.InputKindFilterTaskID.ReferenceIDString()] = tc.inputValue
			}

			result, _, err := inspectiontest.RunInspectionTask(ctx, InputKindFilterTask, inspectioncore_contract.TaskModeRun, inputMap)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrSub, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantResult, result, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("RunInspectionTask() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
