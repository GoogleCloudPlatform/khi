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

package coretask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

func dependencyKey(dep Dependency) string {
	switch d := dep.(type) {
	case taskid.PointToPointDescriptor:
		return "ref:" + d.ReferenceID()
	case taskid.FanInDescriptor:
		return "tag:" + d.Tag()
	default:
		return ""
	}
}

// verifyDataDependencyDeclared verifies that the given dependency descriptor was declared
// in the running task's dependencies and is not an OrderOnly dependency.
func verifyDataDependencyDeclared(ctx context.Context, dep Dependency) {
	deps, err := khictx.GetValue(ctx, core_contract.TaskDependenciesContextKey)
	if err != nil || deps == nil {
		return
	}
	targetKey := dependencyKey(dep)
	if targetKey == "" {
		return
	}

	for _, declared := range deps {
		if dependencyKey(declared) == targetKey {
			if declared.DescriptorKind() == taskid.EdgeKindOrderOnly {
				panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("cannot get task result for order-only dependency: %s", targetKey)))
			}
			return
		}
	}

	panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("undeclared task dependency access: %s", targetKey)))
}

// GetTaskResult retrieves the result of a previously executed required task.
// Panics if the dependency is undeclared, is order-only, or the result is missing.
func GetTaskResult[T any](ctx context.Context, reference taskid.TaskReference[T]) T {
	verifyDataDependencyDeclared(ctx, reference)
	taskResults := khictx.MustGetValue(ctx, core_contract.TaskResultMapContextKey)
	result, found := typedmap.Get(taskResults, typedmap.NewTypedKey[T](reference.ReferenceIDString()))
	if !found {
		var sb strings.Builder
		for _, key := range taskResults.Keys() {
			sb.WriteString("* ")
			sb.WriteString(key)
			sb.WriteByte('\n')
		}
		panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("task result for %s isn't available. Did you add it in the task dependency?\nAvailable task results:\n%s", reference.ReferenceIDString(), sb.String())))
	}
	return result
}

// GetOptionalTaskResult retrieves the result from an optional previously executed task.
// If the task was not included in the execution graph, it safely returns (zeroValue, false).
// If the task was bound in the graph but the result is missing, it panics.
func GetOptionalTaskResult[T any](ctx context.Context, reference taskid.TaskReference[T]) (T, bool) {
	verifyDataDependencyDeclared(ctx, reference)
	refID := reference.ReferenceIDString()
	graphMetadata := khictx.MustGetValue(ctx, core_contract.TaskGraphMetadataContextKey)
	if !graphMetadata.IsBound(refID) {
		var zero T
		return zero, false
	}

	taskResults := khictx.MustGetValue(ctx, core_contract.TaskResultMapContextKey)
	result, found := typedmap.Get(taskResults, typedmap.NewTypedKey[T](refID))
	if !found {
		panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("optional task %s was bound in DAG but result is missing", refID)))
	}
	return result, true
}

// GetTaskResultsWithTag retrieves all results of tasks providing the given tag as a slice.
// Producer task results are returned in deterministic order.
// If no tasks match the tag, an empty slice is returned.
// Panics if the dependency is undeclared, is order-only, or task graph metadata is not available.
func GetTaskResultsWithTag[T any](ctx context.Context, tagReference TagReference[T]) []T {
	verifyDataDependencyDeclared(ctx, tagReference)
	graphMetadata := khictx.MustGetValue(ctx, core_contract.TaskGraphMetadataContextKey)
	taskResults := khictx.MustGetValue(ctx, core_contract.TaskResultMapContextKey)

	boundRefIDs := graphMetadata.BoundReferenceIDsWithTag(tagReference.Tag())
	results := make([]T, 0, len(boundRefIDs))
	for _, refID := range boundRefIDs {
		res, found := typedmap.Get(taskResults, typedmap.NewTypedKey[T](refID))
		if !found {
			panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("task %s providing tag %s result missing", refID, tagReference.Tag())))
		}
		results = append(results, res)
	}
	return results
}

// WrapErrorWithTaskInformation annotates given error with the current task information.
func WrapErrorWithTaskInformation(ctx context.Context, err error) error {
	taskID := khictx.MustGetValue(ctx, core_contract.TaskImplementationIDContextKey)
	errorMessage := fmt.Sprintf("An error occurred in task `%s`", taskID.String())
	return errors.Join(errors.New(errorMessage), err)
}
