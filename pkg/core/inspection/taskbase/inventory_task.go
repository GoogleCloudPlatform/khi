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

// Inventory related tasks defined in this file provides a framework for discovering and merging inventory data from various sources.
//
// In many inspection scenarios, it's necessary to associate information across different log sources.
// For example, a log might contain an IP address, while another log maps that IP to a specific VM or container name.
// However, the availability of these log sources is not always guaranteed, and consumers of this inventory
// data should not need to be aware of the specific tasks that provide it.
//
// This framework introduces two main components to address this:
//
//  1. DiscoveryTask: A task responsible for extracting a inventory map from a single data source.
//     Providers of a discovery task must ensure it is added to the task graph when a task that may require its
//     data is included. This is achieved by using the coretask.NewSubsequentTaskRefsTaskLabel, which links the
//     discovery task to the merger task.
//
//  2. InventoryTask: A task that aggregates the results from all relevant DiscoveryTasks.
//     Consumers can simply depend on this single merger task to access the complete, consolidated inventory map
//     without needing to know about the individual discovery tasks.
//
// This approach decouples data consumers from data providers, allowing for a flexible and extensible inspection system.
package inspectiontaskbase

import (
	"context"
	"log/slog"
	"sync"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InventoryTaskBuilder builds a inventory task and discovery tasks.
// Inventory task merges information found in logs from multiple discovery tasks.
type InventoryTaskBuilder[T any] struct {
	mu                sync.Mutex
	inventoryTaskID   taskid.TaskImplementationID[T]
	discoveryTaskRefs []taskid.TaskReference[T]
}

func NewInventoryTaskBuilder[T any](inventoryTaskID taskid.TaskImplementationID[T]) *InventoryTaskBuilder[T] {
	return &InventoryTaskBuilder[T]{
		inventoryTaskID: inventoryTaskID,
	}
}

// InventoryTask builds a inventory task with given merger strategy.
func (s *InventoryTaskBuilder[T]) InventoryTask(strategy InventoryMergerStrategy[T]) coretask.Task[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return NewInspectionTask(s.inventoryTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (T, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return *new(T), nil
		}
		discoveryResults := make([]T, 0, len(s.discoveryTaskRefs))
		for _, ref := range s.discoveryTaskRefs {
			r, found := coretask.GetTaskResultOptional(ctx, ref)
			if found {
				discoveryResults = append(discoveryResults, r)
			} else {
				slog.DebugContext(ctx, "discovery result not provided", "taskRef", ref.ReferenceIDString())
			}
		}
		return strategy.Merge(discoveryResults)
	})
}

// DiscoveryTask builds a discovery task the returned value from discovery tasks are aggregated in inventory task
func (s *InventoryTaskBuilder[T]) DiscoveryTask(taskID taskid.TaskImplementationID[T], dependencies []taskid.UntypedTaskReference, taskFunc ProgressReportableInspectionTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.Task[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	inventoryTaskID := s.inventoryTaskID.Ref()
	labelOpts = append(labelOpts, coretask.NewSubsequentTaskRefsTaskLabel(inventoryTaskID))

	found := false
	for _, ref := range s.discoveryTaskRefs {
		if ref.ReferenceIDString() == taskID.ReferenceIDString() {
			found = true
			break
		}
	}
	if !found {
		s.discoveryTaskRefs = append(s.discoveryTaskRefs, taskID.Ref())
	}

	return NewProgressReportableInspectionTask(taskID, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (T, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return *new(T), nil
		}
		return taskFunc(ctx, taskMode, progress)
	}, labelOpts...)
}

// InventoryMergerStrategy defines the strategy how the generated InventoryTask merges results received from multiple discovery tasks.
type InventoryMergerStrategy[T any] interface {

	// Merge defines the logic to combine multiple results from various InventoryDiscoveryTasks
	// into a single, consolidated result.
	Merge(results []T) (T, error)
}
