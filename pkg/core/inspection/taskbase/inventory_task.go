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

// Inventory tasks defined in this file provide a framework for discovering and merging inventory data from various sources.
//
// In many inspection scenarios, it's necessary to associate information across different log sources.
// For example, a log might contain an IP address, while another log maps that IP to a specific VM or container name.
//
// This framework introduces a demand-driven approach:
//  1. Discovery Tasks: Extract inventory data from individual sources and publish it via coretask.ProvidesTag(tag).
//  2. Inventory Tasks: Demand-driven aggregators created via NewInventoryTask, which use tag references
//     (coretask.FromActiveFeatures) to pull in and merge data only from producers whose prerequisite data sources are active.
package inspectiontaskbase

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// NewInventoryTask creates an inventory task that dynamically discovers and aggregates outputs
// from all producer tasks that provide the specified tag within active features.
func NewInventoryTask[T any, R any](
	id taskid.TaskImplementationID[R],
	tag coretask.Tag[T],
	mergeFunc func(results []T) (R, error),
	labelOpts ...coretask.LabelOpt,
) coretask.Task[R] {
	tagRef := tag.Ref(coretask.FromActiveFeatures)
	return NewInspectionTask(
		id,
		[]coretask.Dependency{tagRef},
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (R, error) {
			if taskMode == inspectioncore_contract.TaskModeDryRun {
				var zero R
				return zero, nil
			}
			results := coretask.GetTaskResultsWithTag(ctx, tagRef)
			return mergeFunc(results)
		},
		labelOpts...,
	)
}
