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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestCachedTask(t *testing.T) {
	testCases := []struct {
		name       string
		taskID     taskid.TaskImplementationID[string]
		run        func(ctx context.Context, task coretask.Task[string]) context.Context
		wantResult []CacheableTaskResult[string]
	}{
		{
			name:   "caches across runs and across inspections in GlobalSharedMap",
			taskID: taskid.NewDefaultImplementationID[string]("global-cache-test"),
			run: func(ctx context.Context, task coretask.Task[string]) context.Context {
				_, _, _ = inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{})
				_, _, _ = inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{})

				nextInspCtx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
				globalSharedMap := khictx.MustGetValue(ctx, inspectioncore_contract.GlobalSharedMap)
				nextInspCtx = khictx.WithValue(nextInspCtx, inspectioncore_contract.GlobalSharedMap, globalSharedMap)

				_, _, _ = inspectiontest.RunInspectionTask(nextInspCtx, task, inspectioncore_contract.TaskModeRun, map[string]any{})
				return nextInspCtx
			},
			wantResult: []CacheableTaskResult[string]{
				{Value: "", DependencyDigest: ""},
				{Value: "res", DependencyDigest: "digest"},
				{Value: "res", DependencyDigest: "digest"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prevValues := []CacheableTaskResult[string]{}
			task := NewGlobalCachedTask(tc.taskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, prevValue CacheableTaskResult[string]) (CacheableTaskResult[string], error) {
				prevValues = append(prevValues, prevValue)
				return CacheableTaskResult[string]{
					Value:            "res",
					DependencyDigest: "digest",
				}, nil
			})

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_ = tc.run(ctx, task)

			if diff := cmp.Diff(tc.wantResult, prevValues); diff != "" {
				t.Errorf("prevValues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInspectionCachedTask(t *testing.T) {
	testCases := []struct {
		name       string
		taskID     taskid.TaskImplementationID[string]
		run        func(ctx context.Context, task coretask.Task[string]) context.Context
		wantResult []CacheableTaskResult[string]
	}{
		{
			name:   "caches within same inspection but resets on new inspection in InspectionSharedMap",
			taskID: taskid.NewDefaultImplementationID[string]("inspection-cache-test"),
			run: func(ctx context.Context, task coretask.Task[string]) context.Context {
				_, _, _ = inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{})
				_, _, _ = inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{})

				nextInspCtx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
				globalSharedMap := khictx.MustGetValue(ctx, inspectioncore_contract.GlobalSharedMap)
				nextInspCtx = khictx.WithValue(nextInspCtx, inspectioncore_contract.GlobalSharedMap, globalSharedMap)

				_, _, _ = inspectiontest.RunInspectionTask(nextInspCtx, task, inspectioncore_contract.TaskModeRun, map[string]any{})
				return nextInspCtx
			},
			wantResult: []CacheableTaskResult[string]{
				{Value: "", DependencyDigest: ""},
				{Value: "res", DependencyDigest: "digest"},
				{Value: "", DependencyDigest: ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prevValues := []CacheableTaskResult[string]{}
			task := NewInspectionCachedTask(tc.taskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, prevValue CacheableTaskResult[string]) (CacheableTaskResult[string], error) {
				prevValues = append(prevValues, prevValue)
				return CacheableTaskResult[string]{
					Value:            "res",
					DependencyDigest: "digest",
				}, nil
			})

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_ = tc.run(ctx, task)

			if diff := cmp.Diff(tc.wantResult, prevValues); diff != "" {
				t.Errorf("prevValues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
