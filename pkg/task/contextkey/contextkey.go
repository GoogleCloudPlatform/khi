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

package task_contextkey

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// TaskResultMapContextKey is the key to get the result of each tasks run before.
var TaskResultMapContextKey = typedmap.NewTypedKey[*typedmap.TypedMap]("khi.google.com/task-result-map")

// TaskImplementationIDContextKey is the key to get the current task implementation ID.
var TaskImplementationIDContextKey = typedmap.NewTypedKey[taskid.UntypedTaskImplementationID]("khi.google.com/task-implementation-id")
