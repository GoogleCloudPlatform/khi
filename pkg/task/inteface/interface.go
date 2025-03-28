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

package task_interface

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// TaskRunner receives the runnable DefinitionSet and run tasks with topological sorted order.
type TaskRunner interface {
	Run(ctx context.Context) error
	Wait() <-chan interface{}
	Result() (*typedmap.ReadonlyTypedMap, error)
}
