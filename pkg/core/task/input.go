// Copyright 2026 Google LLC
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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// activeDefinedTaskKey is the context key for the implementation ID of the running task defined with Define.
type activeDefinedTaskKey struct{}

// withActiveDefinedTask marks ctx as running the task defined with Define identified by owner.
func withActiveDefinedTask(ctx context.Context, owner taskid.UntypedTaskImplementationID) context.Context {
	return context.WithValue(ctx, activeDefinedTaskKey{}, owner.String())
}

// hasActiveDefinedTask reports whether ctx is running a task defined with Define.
func hasActiveDefinedTask(ctx context.Context) bool {
	_, found := ctx.Value(activeDefinedTaskKey{}).(string)
	return found
}

// mustMatchActiveDefinedTask panics unless ctx is running the task that declared the handle.
// A handle read from another task would bypass that task's input declarations.
func mustMatchActiveDefinedTask(ctx context.Context, owner taskid.UntypedTaskImplementationID, input string) {
	active, found := ctx.Value(activeDefinedTaskKey{}).(string)
	if !found {
		panic(fmt.Sprintf("input %s declared by task %s is read outside of a task defined with Define", input, owner))
	}
	if active != owner.String() {
		panic(fmt.Sprintf("input %s declared by task %s is read from task %s; handles can only be read by the task that declared them", input, owner, active))
	}
}

// Input is a handle to a required input declared with Use.
type Input[T any] struct {
	owner taskid.UntypedTaskImplementationID
	ref   taskid.TaskReference[T]
}

// Get returns the result of the producer task.
func (in Input[T]) Get(ctx context.Context) T {
	mustMatchActiveDefinedTask(ctx, in.owner, dependencyKey(in.ref))
	return lookupTaskResult(ctx, in.ref)
}

// OptionalInput is a handle to an optional input declared with UseOptional.
type OptionalInput[T any] struct {
	owner taskid.UntypedTaskImplementationID
	ref   taskid.TaskReference[T]
}

// Get returns the result of the producer task and true, or the zero value and false when the producer is not in the graph.
func (in OptionalInput[T]) Get(ctx context.Context) (T, bool) {
	mustMatchActiveDefinedTask(ctx, in.owner, dependencyKey(in.ref))
	return lookupOptionalTaskResult(ctx, in.ref)
}

// TagInput is a handle to a fan-in input declared with UseTag.
type TagInput[T any] struct {
	owner taskid.UntypedTaskImplementationID
	ref   TagReference[T]
}

// Get returns the results of all producer tasks bound to the tag in deterministic order.
func (in TagInput[T]) Get(ctx context.Context) []T {
	mustMatchActiveDefinedTask(ctx, in.owner, dependencyKey(in.ref))
	return lookupTaskResultsWithTag(ctx, in.ref)
}
