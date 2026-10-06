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

package inspectiontaskbase

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestDefineCachedTask(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[string]("source")
	taskID := taskid.NewDefaultImplementationID[int]("cached-task")

	newTask := func(scope CacheScope, computeCount *int, failFirst bool) coretask.Task[int] {
		return DefineCachedTask(taskID, func(b *coretask.Binder) CachedTaskSpec[int] {
			source := coretask.Use(b, sourceTaskID.Ref())
			return CachedTaskSpec[int]{
				Scope:       scope,
				InputDigest: source.Get,
				Compute: func(ctx context.Context) (int, error) {
					*computeCount++
					if failFirst && *computeCount == 1 {
						return 0, errors.New("simulated compute failure")
					}
					return *computeCount, nil
				},
			}
		})
	}

	testCases := []struct {
		name             string
		scope            CacheScope
		secondInput      string
		newInspection    bool
		failFirst        bool
		wantFirstErr     bool
		want             []int
		wantComputeCount int
	}{
		{
			name:             "inspection scope, same input in the same inspection",
			scope:            CacheScopeInspection,
			secondInput:      "input-a",
			newInspection:    false,
			failFirst:        false,
			wantFirstErr:     false,
			want:             []int{1, 1},
			wantComputeCount: 1,
		},
		{
			name:             "inspection scope, different input",
			scope:            CacheScopeInspection,
			secondInput:      "input-b",
			newInspection:    false,
			failFirst:        false,
			wantFirstErr:     false,
			want:             []int{1, 2},
			wantComputeCount: 2,
		},
		{
			name:             "inspection scope, same input in a new inspection",
			scope:            CacheScopeInspection,
			secondInput:      "input-a",
			newInspection:    true,
			failFirst:        false,
			wantFirstErr:     false,
			want:             []int{1, 2},
			wantComputeCount: 2,
		},
		{
			name:             "global scope, same input in a new inspection",
			scope:            CacheScopeGlobal,
			secondInput:      "input-a",
			newInspection:    true,
			failFirst:        false,
			wantFirstErr:     false,
			want:             []int{1, 1},
			wantComputeCount: 1,
		},
		{
			name:             "compute error is not cached",
			scope:            CacheScopeInspection,
			secondInput:      "input-a",
			newInspection:    false,
			failFirst:        true,
			wantFirstErr:     true,
			want:             []int{2},
			wantComputeCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			computeCount := 0
			task := newTask(tc.scope, &computeCount, tc.failFirst)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			res1, _, err1 := inspectiontest.Run(t, ctx, task, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(sourceTaskID.Ref(), "input-a"),
			)
			if tc.wantFirstErr {
				if err1 == nil {
					t.Fatalf("first run expected error, got nil")
				}
			} else if err1 != nil {
				t.Fatalf("first run failed: %v", err1)
			}

			secondCtx := ctx
			if tc.newInspection {
				secondCtx = inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
				globalSharedMap := khictx.MustGetValue(ctx, inspectioncore.GlobalSharedMap)
				secondCtx = khictx.WithValue(secondCtx, inspectioncore.GlobalSharedMap, globalSharedMap)
			}

			res2, _, err2 := inspectiontest.Run(t, secondCtx, task, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(sourceTaskID.Ref(), tc.secondInput),
			)
			if err2 != nil {
				t.Fatalf("second run failed: %v", err2)
			}

			var gotResults []int
			if !tc.wantFirstErr {
				gotResults = append(gotResults, res1)
			}
			gotResults = append(gotResults, res2)

			if diff := cmp.Diff(tc.want, gotResults); diff != "" {
				t.Errorf("results mismatch (-want +got):\n%s", diff)
			}
			if computeCount != tc.wantComputeCount {
				t.Errorf("compute count = %d, want %d", computeCount, tc.wantComputeCount)
			}
		})
	}

	t.Run("declares required input on source reference", func(t *testing.T) {
		task := newTask(CacheScopeInspection, new(int), false)
		wantInputs := []string{"required source"}
		if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}
	})
}
