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
	"slices"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestInventoryTask(t *testing.T) {
	inventoryTag := coretask.NewTag[map[string]struct{}]("test-inventory-tag")
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test-merger")

	nop := func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	}

	discovery1ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-1-parent")
	discovery1ParentTask := NewInspectionTask(discovery1ParentTaskID, []coretask.Dependency{}, nop)

	discovery2ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-2-parent")
	discovery2ParentTask := NewInspectionTask(discovery2ParentTaskID, []coretask.Dependency{}, nop)

	discovery1ID := taskid.NewDefaultImplementationID[map[string]struct{}]("discovery-1")
	discovery1 := NewInspectionTask(
		discovery1ID,
		[]coretask.Dependency{discovery1ParentTaskID.Ref()},
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
			return map[string]struct{}{"foo": {}}, nil
		},
		coretask.ProvidesTag(inventoryTag, coretask.WithTagPriority(10)),
	)

	discovery2ID := taskid.NewDefaultImplementationID[map[string]struct{}]("discovery-2")
	discovery2 := NewInspectionTask(
		discovery2ID,
		[]coretask.Dependency{discovery2ParentTaskID.Ref()},
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
			return map[string]struct{}{"bar": {}}, nil
		},
		coretask.ProvidesTag(inventoryTag),
	)

	mergerTask := NewInventoryTask(
		mergerTaskID,
		inventoryTag,
		func(results []map[string]struct{}) (map[string]struct{}, error) {
			result := make(map[string]struct{})
			for _, r := range results {
				for k := range r {
					result[k] = struct{}{}
				}
			}
			return result, nil
		},
	)

	cyclicDiscoveryTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("cyclic-discovery")
	cyclicDiscoveryTask := NewInspectionTask(
		cyclicDiscoveryTaskID,
		[]coretask.Dependency{mergerTaskID.Ref()},
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
			return map[string]struct{}{"cyclic": {}}, nil
		},
		coretask.ProvidesTag(inventoryTag, coretask.WithTagPriority(100)),
	)

	defaultAvailableTasks := []coretask.UntypedTask{
		mergerTask,
		discovery1,
		discovery2,
		discovery1ParentTask,
		discovery2ParentTask,
	}

	testCases := []struct {
		name           string
		availableTasks []coretask.UntypedTask
		userTaskDeps   []coretask.Dependency
		wantMap        map[string]struct{}
	}{
		{
			name:           "provided from single discovery task when only parent 1 is active",
			availableTasks: defaultAvailableTasks,
			userTaskDeps:   []coretask.Dependency{mergerTaskID.Ref(), discovery1ParentTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
			},
		},
		{
			name:           "provided from multiple discovery tasks when both parent 1 and 2 are active",
			availableTasks: defaultAvailableTasks,
			userTaskDeps:   []coretask.Dependency{mergerTaskID.Ref(), discovery1ParentTaskID.Ref(), discovery2ParentTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
				"bar": {},
			},
		},
		{
			name:           "provided from no discovery tasks when neither parent is active",
			availableTasks: defaultAvailableTasks,
			userTaskDeps:   []coretask.Dependency{mergerTaskID.Ref()},
			wantMap:        map[string]struct{}{},
		},
		{
			name:           "prunes circular dependency created by selected cyclic task so merger only includes non-cyclic discovery tasks",
			availableTasks: append(slices.Clone(defaultAvailableTasks), cyclicDiscoveryTask),
			userTaskDeps:   []coretask.Dependency{mergerTaskID.Ref(), discovery1ParentTaskID.Ref(), cyclicDiscoveryTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("user-" + tc.name)
			userTask := NewInspectionTask(
				userTaskID,
				tc.userTaskDeps,
				func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
					return coretask.GetTaskResult(ctx, mergerTaskID.Ref()), nil
				},
			)

			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			gotDryRunMap, _, err := inspectiontest.RunInspectionTaskWithDependency(
				dryRunCtx,
				userTask,
				tc.availableTasks,
				inspectioncore_contract.TaskModeDryRun,
				map[string]any{},
			)
			if err != nil {
				t.Fatalf("RunInspectionTaskWithDependency() dry run error: %v", err)
			}
			if diff := cmp.Diff(map[string]struct{}(nil), gotDryRunMap); diff != "" {
				t.Errorf("merger task dry run result mismatch (-want +got):\n%s", diff)
			}

			runCtx := inspectiontest.NextRunTaskContext(t.Context(), dryRunCtx)
			gotMap, _, err := inspectiontest.RunInspectionTaskWithDependency(
				runCtx,
				userTask,
				tc.availableTasks,
				inspectioncore_contract.TaskModeRun,
				map[string]any{},
			)
			if err != nil {
				t.Fatalf("RunInspectionTaskWithDependency() run error: %v", err)
			}
			if diff := cmp.Diff(tc.wantMap, gotMap); diff != "" {
				t.Errorf("merger task result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
