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

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type testSimpleStringRelationshipMergerTaskSetting struct {
	idSource *RelationshipTaskIDSource[map[string]struct{}]
}

// IDSource implements RelationshipMergerTaskSetting.
func (t *testSimpleStringRelationshipMergerTaskSetting) IDSource() *RelationshipTaskIDSource[map[string]struct{}] {
	return t.idSource
}

// Merge implements RelationshipMergerTaskSetting.
func (t *testSimpleStringRelationshipMergerTaskSetting) Merge(results []map[string]struct{}) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for _, r := range results {
		for k := range r {
			result[k] = struct{}{}
		}
	}
	return result, nil
}

var _ RelationshipMergerTaskSetting[map[string]struct{}] = (*testSimpleStringRelationshipMergerTaskSetting)(nil)

// TestRelationshipTaskIDSource tests the basic functionality of the RelationshipTaskIDSource.
// It verifies that:
// - GenerateDefaultRelationshipDiscoveryTaskID correctly creates a task ID.
// - Attempting to generate a task ID with the same reference string multiple times does not result in duplicate entries.
func TestRelationshipTaskIDSource(t *testing.T) {
	mergerTaskID := taskid.NewDefaultImplementationID[struct{}]("test")
	ids := NewRelationshipTaskIDSource(mergerTaskID)

	wantID := taskid.NewDefaultImplementationID[struct{}]("discovery")
	gotID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery")

	if diff := cmp.Diff(wantID.String(), gotID.String()); diff != "" {
		t.Errorf("GenerateDefaultRelationshipDiscoveryTaskID() mismatch (-want +got):\n%s", diff)
	}

	ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery")
	if len(ids.discoveryTaskRefs) != 1 {
		t.Errorf("Expected discoveryTaskRefs to contain 1 element after generating the same ID twice, got %d", len(ids.discoveryTaskRefs))
	}
}

// TestRelationshipTask_ProvidedFromSingleDiscoveryTask tests a scenario where the merger task
// receives data from only one of two available discovery tasks.
// This is because the main user task only depends on the parent of the first discovery task.
// The test verifies that only the result from the first discovery task ("foo") is present in the final merged map and the task dependency topology doesn't add the discovery-2 task not intentionally.
func TestRelationshipTask_ProvidedFromSingleDiscoveryTask(t *testing.T) {
	nop := func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	}
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test")
	ids := NewRelationshipTaskIDSource(mergerTaskID)
	mergerTask := NewRelationshipMergerTask(&testSimpleStringRelationshipMergerTaskSetting{ids})
	discovery1ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-1")
	discovery2ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-2")
	discovery1ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-1-parent")
	discovery1ParentTask := NewInspectionTask(discovery1ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery1ID.Ref()))
	discovery1 := NewRelationshipDiscoveryTask(discovery1ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"foo": {},
		}, nil
	})
	discovery2ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-2-parent")
	discovery2ParentTask := NewInspectionTask(discovery2ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery2ID.Ref()))
	discovery2 := NewRelationshipDiscoveryTask(discovery2ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"bar": {},
		}, nil
	})
	userTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("user")

	userTask := NewInspectionTask(userTaskID, []taskid.UntypedTaskReference{mergerTaskID.Ref(), discovery1ParentTaskID.Ref()}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
		return coretask.GetTaskResult(ctx, mergerTaskID.Ref()), nil
	})

	wantMap := map[string]struct{}{
		"foo": {},
	}
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotMap, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, userTask, []coretask.UntypedTask{mergerTask, discovery1, discovery2, discovery1ParentTask, discovery2ParentTask}, inspectioncore_contract.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("running merger task failed with error: %v", err)
	}
	if diff := cmp.Diff(wantMap, gotMap); diff != "" {
		t.Errorf("merger task result mismatch (-want +got):\n%s", diff)
	}
}

// TestRelationshipTask_ProvidedFromMultipleDiscoveryTask tests a scenario where the merger task
// receives and merges data from multiple discovery tasks.
// This is because the main user task depends on the parents of both discovery tasks.
// The test verifies that the results from both discovery tasks ("foo" and "bar") are present in the final merged map.
func TestRelationshipTask_ProvidedFromMultipleDiscoveryTask(t *testing.T) {
	nop := func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	}
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test")
	ids := NewRelationshipTaskIDSource(mergerTaskID)
	mergerTask := NewRelationshipMergerTask(&testSimpleStringRelationshipMergerTaskSetting{ids})
	discovery1ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-1")
	discovery2ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-2")
	discovery1ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-1-parent")
	discovery1ParentTask := NewInspectionTask(discovery1ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery1ID.Ref()))
	discovery1 := NewRelationshipDiscoveryTask(discovery1ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"foo": {},
		}, nil
	})
	discovery2ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-2-parent")
	discovery2ParentTask := NewInspectionTask(discovery2ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery2ID.Ref()))
	discovery2 := NewRelationshipDiscoveryTask(discovery2ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"bar": {},
		}, nil
	})
	userTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("user")

	userTask := NewInspectionTask(userTaskID, []taskid.UntypedTaskReference{mergerTaskID.Ref(), discovery1ParentTaskID.Ref(), discovery2ParentTaskID.Ref()}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
		return coretask.GetTaskResult(ctx, mergerTaskID.Ref()), nil
	})

	wantMap := map[string]struct{}{
		"foo": {},
		"bar": {},
	}
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotMap, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, userTask, []coretask.UntypedTask{mergerTask, discovery1, discovery2, discovery1ParentTask, discovery2ParentTask}, inspectioncore_contract.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("running merger task failed with error: %v", err)
	}
	if diff := cmp.Diff(wantMap, gotMap); diff != "" {
		t.Errorf("merger task result mismatch (-want +got):\n%s", diff)
	}
}

// TestRelationshipTask_ProvidedFromNoDiscoveryTask tests a scenario where the merger task receives no data.
// This is because the main user task does not depend on any of the discovery tasks' parents.
// The test verifies that the final merged map is empty.
func TestRelationshipTask_ProvidedFromNoDiscoveryTask(t *testing.T) {
	nop := func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	}
	mergerTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("test")
	ids := NewRelationshipTaskIDSource(mergerTaskID)
	mergerTask := NewRelationshipMergerTask(&testSimpleStringRelationshipMergerTaskSetting{ids})
	discovery1ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-1")
	discovery2ID := ids.GenerateDefaultRelationshipDiscoveryTaskID("discovery-2")
	discovery1ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-1-parent")
	discovery1ParentTask := NewInspectionTask(discovery1ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery1ID.Ref()))
	discovery1 := NewRelationshipDiscoveryTask(discovery1ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"foo": {},
		}, nil
	})
	discovery2ParentTaskID := taskid.NewDefaultImplementationID[struct{}]("discovery-2-parent")
	discovery2ParentTask := NewInspectionTask(discovery2ParentTaskID, []taskid.UntypedTaskReference{}, nop, coretask.NewSubsequentTaskRefsTaskLabel(discovery2ID.Ref()))
	discovery2 := NewRelationshipDiscoveryTask(discovery2ID, ids, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (map[string]struct{}, error) {
		return map[string]struct{}{
			"bar": {},
		}, nil
	})
	userTaskID := taskid.NewDefaultImplementationID[map[string]struct{}]("user")

	userTask := NewInspectionTask(userTaskID, []taskid.UntypedTaskReference{mergerTaskID.Ref()}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (map[string]struct{}, error) {
		return coretask.GetTaskResult(ctx, mergerTaskID.Ref()), nil
	})

	wantMap := map[string]struct{}{}
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotMap, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, userTask, []coretask.UntypedTask{mergerTask, discovery1, discovery2, discovery1ParentTask, discovery2ParentTask}, inspectioncore_contract.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("running merger task failed with error: %v", err)
	}
	if diff := cmp.Diff(wantMap, gotMap); diff != "" {
		t.Errorf("merger task result mismatch (-want +got):\n%s", diff)
	}
}
