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

package coreinspection_test

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestRunContextOptionFromValue(t *testing.T) {
	t.Parallel()
	var testKey = typedmap.NewTypedKey[string]("foo")
	const testValue = "test-value"

	option := coreinspection.RunContextOptionFromValue(testKey, testValue)

	ctx, err := option(context.Background(), inspectioncore_contract.TaskModeRun)
	if err != nil {
		t.Fatalf("option() failed: %v", err)
	}

	gotValue, err := khictx.GetValue(ctx, testKey)
	if err != nil {
		t.Errorf("value not found in context %v", err)
	}

	if diff := cmp.Diff(testValue, gotValue); diff != "" {
		t.Errorf("unexpected value (-want +got):\n%s", diff)
	}
}

func TestRunContextOptionFromFunc(t *testing.T) {
	t.Parallel()
	var testKey = typedmap.NewTypedKey[string]("foo")
	const testValue = "test-value"
	var testErr = errors.New("test-error")

	tests := []struct {
		name      string
		f         func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (string, error)
		wantValue string
		wantErr   error
	}{
		{
			name: "success",
			f: func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (string, error) {
				return testValue, nil
			},
			wantValue: testValue,
		},
		{
			name: "error",
			f: func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (string, error) {
				return "", testErr
			},
			wantErr: testErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			option := coreinspection.RunContextOptionFromFunc(testKey, tt.f)
			ctx, err := option(context.Background(), inspectioncore_contract.TaskModeRun)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("option() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			gotValue, err := khictx.GetValue(ctx, testKey)
			if err != nil {
				t.Errorf("value not found in context %v", err)
			}

			if diff := cmp.Diff(tt.wantValue, gotValue); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunContextOptionArrayElementFromValue(t *testing.T) {
	t.Parallel()
	var testKey = typedmap.NewTypedKey[*[]string]("foo")
	const testValue1 = "test-value-1"
	const testValue2 = "test-value-2"

	tests := []struct {
		name      string
		initial   *[]string
		addValue  string
		wantValue *[]string
		wantErr   error
	}{
		{
			name:      "add to nil",
			initial:   nil,
			addValue:  testValue1,
			wantValue: &[]string{testValue1},
		},
		{
			name:      "add to existing",
			initial:   &[]string{testValue1},
			addValue:  testValue2,
			wantValue: &[]string{testValue1, testValue2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.initial != nil {
				ctx = khictx.WithValue(ctx, testKey, tt.initial)
			}

			option := coreinspection.RunContextOptionArrayElementFromValue(testKey, tt.addValue)
			ctx, err := option(ctx, inspectioncore_contract.TaskModeRun)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("option() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			gotValue, err := khictx.GetValue(ctx, testKey)
			if err != nil {
				t.Errorf("value not found in context %v", err)
			}

			if diff := cmp.Diff(tt.wantValue, gotValue); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}
