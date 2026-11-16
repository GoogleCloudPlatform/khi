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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// TaskRunner receives the runnable TaskSet and run tasks with topological sorted order.
type TaskRunner interface {
	// Run starts executing the task graph asynchronously.
	Run(ctx context.Context) error
	// Wait returns a channel that closes when task graph execution completes.
	Wait() <-chan interface{}
	// Result returns the final task results map or an error if execution failed or has not finished.
	Result() (*typedmap.ReadonlyTypedMap, error)
	// Tasks returns all tasks in the resolved task set.
	Tasks() []UntypedTask
	// TaskRunStatuses returns a snapshot of the execution status for each task keyed by implementation ID.
	TaskRunStatuses() map[string]TaskRunStatus
	// AddInterceptor registers an interceptor to wrap task execution.
	AddInterceptor(interceptor Interceptor)
}
