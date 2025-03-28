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

package inspection_task_contextkey

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
)

// InspectionTaskMode is the context key to access the execution mode of the inspection task.
var InspectionTaskMode = typedmap.NewTypedKey[inspection_task_interface.InspectionTaskMode]("khi.google.com/inspection/task-mode")

// InspectionTaskInput is the context key to access the input parameters for the inspection task.
// It contains a map of parameter names to their values.
var InspectionTaskInput = typedmap.NewTypedKey[map[string]any]("khi.google.com/inspection/task-input")

// InspectionSharedMap is the context key to access a shared typed map
// that persists across multiple executions of an inspection task.
// This map maintains its values between task executions and does not reset,
// making it useful for caching previous task results or storing graph-specific data.
var InspectionSharedMap = typedmap.NewTypedKey[*typedmap.TypedMap]("khi.google.com/inspection/task-graph-shared-map")

// GlobalSharedMap is the context key to access a shared typed map across any inspection tasks.
var GlobalSharedMap = typedmap.NewTypedKey[*typedmap.TypedMap]("khi.google.com/inspection/global-shared-map")

// InspectionTaskInspectionID is the context key to access the unique identifier for the current inspection.
// This ID remains the same for all runs within a single inspection session.
var InspectionTaskInspectionID = typedmap.NewTypedKey[string]("khi.google.com/inspection/inspection-id")

// InspectionTaskRunID is the context key to access the unique identifier for the current task run.
// A new run ID is generated each time an inspection task is executed, allowing differentiation
// between multiple executions of the same inspection.
var InspectionTaskRunID = typedmap.NewTypedKey[string]("khi.google.com/inspection/task-run-id")

// InspectionRunMetadata is the context key to access the metadata map for the current inspection run.
// This map stores supplementary data beyond the main task results, such as logs and progress information.
// It is expected to be serialized and passed to the frontend for display.
var InspectionRunMetadata = typedmap.NewTypedKey[*typedmap.ReadonlyTypedMap]("khi.google.com/inspection/metadata-map")
