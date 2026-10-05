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

// lookupTaskResult reads the result of a required task from the task result map in ctx.
func lookupTaskResult[T any](ctx context.Context, reference taskid.TaskReference[T]) T {
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

// lookupOptionalTaskResult reads the result of an optional task, returning false when the task is not bound in the graph.
func lookupOptionalTaskResult[T any](ctx context.Context, reference taskid.TaskReference[T]) (T, bool) {
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

// lookupTaskResultsWithTag reads the results of all producers bound to the tag for the running task.
func lookupTaskResultsWithTag[T any](ctx context.Context, tagReference TagReference[T]) []T {
	graphMetadata := khictx.MustGetValue(ctx, core_contract.TaskGraphMetadataContextKey)
	taskResults := khictx.MustGetValue(ctx, core_contract.TaskResultMapContextKey)
	taskImplementationID := khictx.MustGetValue(ctx, core_contract.TaskImplementationIDContextKey)

	boundRefIDs := graphMetadata.BoundReferenceIDsForTaskImplWithTag(taskImplementationID.String(), tagReference.Tag())
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
