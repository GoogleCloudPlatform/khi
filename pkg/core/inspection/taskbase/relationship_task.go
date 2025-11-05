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

// Relationship related tasks defined in this file provides a framework for discovering and merging relational data from various sources.
//
// In many inspection scenarios, it's necessary to associate information across different log sources.
// For example, a log might contain an IP address, while another log maps that IP to a specific VM or container name.
// However, the availability of these log sources is not always guaranteed, and consumers of this relational
// data should not need to be aware of the specific tasks that provide it.
//
// This framework introduces two main components to address this:
//
//  1. RelationshipDiscoveryTask: A task responsible for extracting a relationship map from a single data source.
//     Providers of a discovery task must ensure it is added to the task graph when a task that may require its
//     data is included. This is achieved by using the coretask.NewSubsequentTaskRefsTaskLabel, which links the
//     discovery task to the merger task.
//
//  2. RelationshipMergerTask: A task that aggregates the results from all relevant RelationshipDiscoveryTasks.
//     Consumers can simply depend on this single merger task to access the complete, consolidated relationship map
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

// RelationshipTaskIDSource manages the task IDs for RelationshipMergerTask and RelationshipDiscoveryTasks,
// maintaining the references between them.
type RelationshipTaskIDSource[T any] struct {
	mu                sync.Mutex
	mergerTaskID      taskid.TaskImplementationID[T]
	discoveryTaskRefs []taskid.TaskReference[T]
}

func NewRelationshipTaskIDSource[T any](mergerTaskID taskid.TaskImplementationID[T]) *RelationshipTaskIDSource[T] {
	return &RelationshipTaskIDSource[T]{
		mergerTaskID: mergerTaskID,
	}
}

func (s *RelationshipTaskIDSource[T]) GenerateDefaultRelationshipDiscoveryTaskID(taskReferenceID string) taskid.TaskImplementationID[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID := taskid.NewDefaultImplementationID[T](taskReferenceID)

	for _, ref := range s.discoveryTaskRefs {
		if ref.ReferenceIDString() == taskID.ReferenceIDString() {
			return taskID
		}
	}

	s.discoveryTaskRefs = append(s.discoveryTaskRefs, taskID.Ref())
	return taskID
}

func (s *RelationshipTaskIDSource[T]) MergerTaskRef() taskid.TaskReference[T] {
	return s.mergerTaskID.Ref()
}

// RelationshipMergerTaskSetting defines the settings for a RelationshipMergerTask.
// It provides the ID source and the logic to merge results from multiple discovery tasks.
type RelationshipMergerTaskSetting[T any] interface {
	// IDSource returns the pointer to the RelationshipTaskIDSource that manages the task IDs.
	IDSource() *RelationshipTaskIDSource[T]

	// Merge defines the logic to combine multiple results from various RelationshipDiscoveryTasks
	// into a single, consolidated result.
	Merge(results []T) (T, error)
}

// NewRelationshipMergerTask creates a task that merges relationship maps from all related RelationshipDiscoveryTasks.
// Consumers of the relationship data can simply depend on this merger task without needing to know about
// the specific discovery tasks. It retrieves results from all discovery tasks registered with the provided IDSource
// and merges them using the logic defined in the RelationshipMergerTaskSetting.
func NewRelationshipMergerTask[T any](setting RelationshipMergerTaskSetting[T]) coretask.Task[T] {
	idSource := setting.IDSource()
	return NewInspectionTask(idSource.mergerTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (T, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return *new(T), nil
		}
		discoveryResults := make([]T, 0, len(idSource.discoveryTaskRefs))
		for _, ref := range idSource.discoveryTaskRefs {
			r, found := coretask.GetTaskResultOptional(ctx, ref)
			if found {
				discoveryResults = append(discoveryResults, r)
			} else {
				slog.DebugContext(ctx, "discovery result not provided", "taskRef", ref.ReferenceIDString())
			}
		}
		return setting.Merge(discoveryResults)
	})
}

// NewRelationshipDiscoveryTask creates a task that discovers relationships from a specific data source,
// such as mapping an IP address to a VM name from a log source. The discovered relationship map is then provided
// to the RelationshipMergerTask.
// Implementers of a discovery task must ensure that it is added to the task graph when its data is potentially
// required. This is enforced by adding task reference to this discovery task to the coretask.NewSubsequentTaskRefsTaskLabel on its parent task, which require this discovery task to depend on its parent when it's included.
func NewRelationshipDiscoveryTask[T any](taskID taskid.TaskImplementationID[T], idSource *RelationshipTaskIDSource[T], dependencies []taskid.UntypedTaskReference, taskFunc ProgressReportableInspectionTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.Task[T] {
	mergerTaskID := idSource.MergerTaskRef()
	labelOpts = append(labelOpts, coretask.NewSubsequentTaskRefsTaskLabel(mergerTaskID))
	return NewProgressReportableInspectionTask(taskID, dependencies, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (T, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return *new(T), nil
		}
		return taskFunc(ctx, taskMode, progress)
	}, labelOpts...)
}
