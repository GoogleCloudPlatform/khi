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
		sharedMap := khictx.MustGetValue(ctx, sharedMapKey)
		cacheKey := typedmap.NewTypedKey[CacheableTaskResult[T]](fmt.Sprintf("cached_result-%s", taskID.String()))
		cachedResult := typedmap.GetOrDefault(sharedMap, cacheKey, CacheableTaskResult[T]{})

		nextCache, err := f(ctx, cachedResult)
		if err != nil {
			return *new(T), err
		}

		typedmap.Set(sharedMap, cacheKey, nextCache)
		return nextCache.Value, nil
	}, labelOpt...)
}

// CacheScope is the lifetime of the value cached by a task defined with DefineCachedTask.
type CacheScope int

const (
	// CacheScopeInspection keeps the cached value in InspectionSharedMap, so it is reused across dry runs and the run of the same inspection.
	// This is the zero value and the default.
	CacheScopeInspection CacheScope = iota
	// CacheScopeGlobal keeps the cached value in GlobalSharedMap, so it is reused across inspections.
	CacheScopeGlobal
)

// sharedMapKey returns the context key of the shared map that stores values cached with the scope.
// It panics for values other than the declared CacheScope constants because they indicate a programming error.
func (s CacheScope) sharedMapKey() typedmap.TypedKey[*typedmap.TypedMap] {
	switch s {
	case CacheScopeInspection:
		return inspectioncore.InspectionSharedMap
	case CacheScopeGlobal:
		return inspectioncore.GlobalSharedMap
	default:
		panic(fmt.Sprintf("unknown CacheScope %d", int(s)))
	}
}

// CachedTaskSpec holds the cache scope and the functions of a task defined with DefineCachedTask.
type CachedTaskSpec[T any] struct {
	// Scope is the lifetime of the cached value. The zero value caches the value per inspection.
	Scope CacheScope
	// InputDigest returns a string that identifies the inputs. Compute runs only when it differs from the digest of the cached value.
	InputDigest func(ctx context.Context) string
	// Compute computes the value from the inputs. An error is returned from the task and is not cached.
	Compute func(ctx context.Context) (T, error)
}

// cachedEntry is the value cached by a task defined with DefineCachedTask together with the digest of its inputs.
type cachedEntry[T any] struct {
	inputDigest string
	value       T
}

// DefineCachedTask defines a task with a Binder that reuses the previously computed value while the digest of its inputs is unchanged.
// bind declares the inputs on the Binder and returns the spec whose functions read them. Only the latest value is kept.
func DefineCachedTask[T any](taskID taskid.TaskImplementationID[T], bind func(b *coretask.Binder) CachedTaskSpec[T], labelOpts ...coretask.LabelOpt) coretask.DefinedTask[T] {
	return coretask.Define(taskID, func(b *coretask.Binder) func(ctx context.Context) (T, error) {
		spec := bind(b)
		sharedMapKey := spec.Scope.sharedMapKey()
		cacheKey := typedmap.NewTypedKey[cachedEntry[T]](fmt.Sprintf("cached-entry-%s", taskID.String()))
		return func(ctx context.Context) (T, error) {
			sharedMap := khictx.MustGetValue(ctx, sharedMapKey)
			inputDigest := spec.InputDigest(ctx)
			if cached, found := typedmap.Get(sharedMap, cacheKey); found && cached.inputDigest == inputDigest {
				return cached.value, nil
			}
			value, err := spec.Compute(ctx)
			if err != nil {
				return *new(T), err
			}
			typedmap.Set(sharedMap, cacheKey, cachedEntry[T]{inputDigest: inputDigest, value: value})
			return value, nil
		}
	}, labelOpts...)
}
