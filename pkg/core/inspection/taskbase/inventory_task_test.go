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
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestInventoryTask(t *testing.T) {
	inventoryTag := coretask.NewTag[map[string]struct{}]("test-inventory-tag")
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test-merger")

	nop := func(b *coretask.Binder) InspectionTaskFunc[struct{}] {
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (struct{}, error) {
			return struct{}{}, nil
		}
	}

	discovery1ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-1-parent")
	discovery1ParentTask := DefineInspectionTask(discovery1ParentTaskID, nop)

	discovery2ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-2-parent")
	discovery2ParentTask := DefineInspectionTask(discovery2ParentTaskID, nop)

	discovery1ID := taskid.NewDefaultImplementationID[map[string]struct{}]("discovery-1")
	discovery1 := DefineInspectionTask(
		discovery1ID,
		func(b *coretask.Binder) InspectionTaskFunc[map[string]struct{}] {
			coretask.After(b, discovery1ParentTaskID.Ref())
			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (map[string]struct{}, error) {
				return map[string]struct{}{"foo": {}}, nil
			}
		},
		coretask.ProvidesTag(inventoryTag, coretask.WithTagPriority(10)),
		coretask.WithFeatureGate(discovery1ParentTaskID.Ref()),
	)

	discovery2ID := taskid.NewDefaultImplementationID[map[string]struct{}]("discovery-2")
	discovery2 := DefineInspectionTask(
		discovery2ID,
		func(b *coretask.Binder) InspectionTaskFunc[map[string]struct{}] {
			coretask.After(b, discovery2ParentTaskID.Ref())
			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (map[string]struct{}, error) {
				return map[string]struct{}{"bar": {}}, nil
			}
		},
		coretask.ProvidesTag(inventoryTag),
		coretask.WithFeatureGate(discovery2ParentTaskID.Ref()),
	)

	mergerTask := DefineInventoryTask(
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
	cyclicDiscoveryTask := DefineInspectionTask(
		cyclicDiscoveryTaskID,
		func(b *coretask.Binder) InspectionTaskFunc[map[string]struct{}] {
			coretask.After(b, mergerTaskID.Ref())
			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (map[string]struct{}, error) {
				return map[string]struct{}{"cyclic": {}}, nil
			}
		},
		coretask.ProvidesTag(inventoryTag, coretask.WithTagPriority(100)),
		coretask.WithFeatureGate(discovery1ParentTaskID.Ref()),
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
		userTaskAfter  []coretask.Dependency
		wantMap        map[string]struct{}
	}{
		{
			name:           "provided from single discovery task when only parent 1 is active",
			availableTasks: defaultAvailableTasks,
			userTaskAfter:  []coretask.Dependency{discovery1ParentTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
			},
		},
		{
			name:           "provided from multiple discovery tasks when both parent 1 and 2 are active",
			availableTasks: defaultAvailableTasks,
			userTaskAfter:  []coretask.Dependency{discovery1ParentTaskID.Ref(), discovery2ParentTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
				"bar": {},
			},
		},
		{
			name:           "provided from no discovery tasks when neither parent is active",
			availableTasks: defaultAvailableTasks,
			userTaskAfter:  []coretask.Dependency{},
			wantMap:        map[string]struct{}{},
		},
		{
			name:           "prunes circular dependency created by selected cyclic task so merger only includes non-cyclic discovery tasks",
			availableTasks: append(slices.Clone(defaultAvailableTasks), cyclicDiscoveryTask),
			userTaskAfter:  []coretask.Dependency{discovery1ParentTaskID.Ref(), cyclicDiscoveryTaskID.Ref()},
			wantMap: map[string]struct{}{
				"foo": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("user-" + tc.name)
			userTask := DefineInspectionTask(
				userTaskID,
				func(b *coretask.Binder) InspectionTaskFunc[map[string]struct{}] {
					merged := coretask.Use(b, mergerTaskID.Ref())
					for _, dep := range tc.userTaskAfter {
						coretask.After(b, dep)
					}
					return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (map[string]struct{}, error) {
						return merged.Get(ctx), nil
					}
				},
			)

			dryRunCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			gotDryRunMap, _, err := inspectiontest.RunInspectionTaskWithDependency(
				dryRunCtx,
				userTask,
				tc.availableTasks,
				inspectioncore.TaskModeDryRun,
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
				inspectioncore.TaskModeRun,
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

func TestInventoryTask_Inputs(t *testing.T) {
	inventoryTag := coretask.NewTag[map[string]struct{}]("test-inventory-tag")
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test-merger")
	mergerTask := DefineInventoryTask(
		mergerTaskID,
		inventoryTag,
		func(results []map[string]struct{}) (map[string]struct{}, error) {
			return nil, nil
		},
	)
	inputs := mergerTask.Inputs()
	if len(inputs) != 1 {
		t.Fatalf("Inputs() count = %d, want 1", len(inputs))
	}
	if inputs[0].Kind != coretask.InputKindTag {
		t.Errorf("inputs[0].Kind = %v, want %v", inputs[0].Kind, coretask.InputKindTag)
	}
	tagRef := inputs[0].Dependency.(taskid.FanInDescriptor)
	if tagRef.Tag() != "test-inventory-tag" {
		t.Errorf("Tag() = %q, want %q", tagRef.Tag(), "test-inventory-tag")
	}
	if tagRef.DescriptorScope() != coretask.FromActiveFeatures {
		t.Errorf("DescriptorScope() = %v, want %v", tagRef.DescriptorScope(), coretask.FromActiveFeatures)
	}
}
