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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// CacheableTaskResult is the combination of the cached value and a digest of its dependency.
type CacheableTaskResult[T any] struct {
	// Value is the value used previous run.
	Value T
	// DependencyDigest is a string representation of digest of its inputs.
	// Task must generate a different value for the different combination of the input and task should compare the current digest generated from the current inputs and the previous value digest, then it should return the previous value only when the digest is not changed.
	DependencyDigest string
}

// CachedTaskFunc computes the next cached result from the result cached by the previous run.
type CachedTaskFunc[T any] func(ctx context.Context, prevValue CacheableTaskResult[T]) (CacheableTaskResult[T], error)

// NewGlobalCachedTask generates a task which can reuse the value from previous runs stored in GlobalSharedMap.
func NewGlobalCachedTask[T any](taskID taskid.TaskImplementationID[T], dependencies []coretask.Dependency, f func(ctx context.Context, prevValue CacheableTaskResult[T]) (CacheableTaskResult[T], error), labelOpt ...coretask.LabelOpt) coretask.Task[T] {
	return newCachedTaskWithSharedMapKey(inspectioncore.GlobalSharedMap, taskID, dependencies, f, labelOpt...)
}

// NewInspectionCachedTask generates a task which can reuse the value from previous runs within the same inspection stored in InspectionSharedMap.
// To clean up resources after inspection, use context.AfterFunc as below:
//
//	inspectionContext := khictx.MustGetValue(ctx, inspectioncore.InspectionContext)
//	context.AfterFunc(inspectionContext, func() {
//		// Dispose allocated resource here.
//	})
func NewInspectionCachedTask[T any](taskID taskid.TaskImplementationID[T], dependencies []coretask.Dependency, f func(ctx context.Context, prevValue CacheableTaskResult[T]) (CacheableTaskResult[T], error), labelOpt ...coretask.LabelOpt) coretask.Task[T] {
	return newCachedTaskWithSharedMapKey(inspectioncore.InspectionSharedMap, taskID, dependencies, f, labelOpt...)
}

func newCachedTaskWithSharedMapKey[T any](sharedMapKey typedmap.TypedKey[*typedmap.TypedMap], taskID taskid.TaskImplementationID[T], dependencies []coretask.Dependency, f func(ctx context.Context, prevValue CacheableTaskResult[T]) (CacheableTaskResult[T], error), labelOpt ...coretask.LabelOpt) coretask.Task[T] {
	return coretask.NewTask(taskID, dependencies, func(ctx context.Context) (T, error) {
		return runCachedTask(ctx, sharedMapKey, taskID, f)
	}, labelOpt...)
}

// runCachedTask passes the previous cached result to f and stores the next result in the shared map.
func runCachedTask[T any](ctx context.Context, sharedMapKey typedmap.TypedKey[*typedmap.TypedMap], taskID taskid.TaskImplementationID[T], f CachedTaskFunc[T]) (T, error) {
	sharedMap := khictx.MustGetValue(ctx, sharedMapKey)
	cacheKey := typedmap.NewTypedKey[CacheableTaskResult[T]](fmt.Sprintf("cached_result-%s", taskID.String()))
	cachedResult := typedmap.GetOrDefault(sharedMap, cacheKey, CacheableTaskResult[T]{})

	nextCache, err := f(ctx, cachedResult)
	if err != nil {
		return *new(T), err
	}

	typedmap.Set(sharedMap, cacheKey, nextCache)
	return nextCache.Value, nil
}

// DefineGlobalCachedTask defines a task with a Binder that can reuse the value from previous runs stored in GlobalSharedMap.
// bind declares the inputs on the Binder and returns the function that computes the next cached result from them.
func DefineGlobalCachedTask[T any](taskID taskid.TaskImplementationID[T], bind func(b *coretask.Binder) CachedTaskFunc[T], labelOpts ...coretask.LabelOpt) coretask.DefinedTask[T] {
	return coretask.Define(taskID, func(b *coretask.Binder) func(ctx context.Context) (T, error) {
		f := bind(b)
		return func(ctx context.Context) (T, error) {
			return runCachedTask(ctx, inspectioncore.GlobalSharedMap, taskID, f)
		}
	}, labelOpts...)
}
