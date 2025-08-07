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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/core/contract/taskid"
)

// GetTaskResult retrieves the result of a previously executed task.
func GetTaskResult[T any](ctx context.Context, reference taskid.TaskReference[T]) T {
	taskResults := khictx.MustGetValue(ctx, core_contract.TaskResultMapContextKey)
	result, found := typedmap.Get(taskResults, typedmap.NewTypedKey[T](reference.ReferenceIDString()))
	if !found {
		availableTaskResults := ""
		for _, key := range taskResults.Keys() {
			availableTaskResults += fmt.Sprintf("* %s\n", key)
		}
		panic(WrapErrorWithTaskInformation(ctx, fmt.Errorf("task result for %s isn't available. Did you add it in the task dependency?", reference.ReferenceIDString())))
	}
	return result
}

// WrapErrorWithTaskInformation annotate given error with the current task information.
func WrapErrorWithTaskInformation(ctx context.Context, err error) error {
	taskID := khictx.MustGetValue(ctx, core_contract.TaskImplementationIDContextKey)
	errorMessage := fmt.Sprintf("An error occurred in task `%s`", taskID.String())
	return errors.Join(errors.New(errorMessage), err)
}
