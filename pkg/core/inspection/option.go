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

package coreinspection

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// RunContextOption modify the given context before passing it to the task runner.
type RunContextOption = func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (context.Context, error)

// RunContextOptionFromValue creates a RunContextOption that adds a static value to the run context.
// The value is added to the context using the provided TypedKey.
func RunContextOptionFromValue[T any](key typedmap.TypedKey[T], value T) RunContextOption {
	return func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (context.Context, error) {
		return khictx.WithValue(ctx, key, value), nil
	}
}

// RunContextOptionFromFunc creates a RunContextOption that adds a dynamically generated value to the context.
// The function `f` is executed to produce the value, which is then added to the run context using the provided TypedKey.
// If `f` returns an error, the option propagation stops and the error is returned.
func RunContextOptionFromFunc[T any](key typedmap.TypedKey[T], f func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (T, error)) RunContextOption {
	return func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (context.Context, error) {
		value, err := f(ctx, mode)
		if err != nil {
			return nil, err
		}
		return khictx.WithValue(ctx, key, value), nil
	}
}

// RunContextOptionArrayElementFromValue creates a RunContextOption that appends a value to an array
// stored in the context under the given TypedKey. If the array does not exist, it initializes an array with the element.
func RunContextOptionArrayElementFromValue[T any](key typedmap.TypedKey[*[]T], value T) RunContextOption {
	return func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (context.Context, error) {
		array, err := khictx.GetValue(ctx, key)
		if err != nil {
			array = &[]T{}
			ctx = khictx.WithValue(ctx, key, array)
		}
		*array = append(*array, value)
		return ctx, nil
	}
}
