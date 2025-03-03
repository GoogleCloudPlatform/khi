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

package task

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// TaskContextIDs is a type of returned value from GetIDsFromTaskContext
type TaskContextIDs struct {
	// InspectionID is an unique ID by each inspections.
	InspectionID string
	// TaskID is the unique ID for the currently executed task.
	TaskID taskid.TaskImplementationId
	// RunID is an unique ID by each run of its task graph.
	RunID string
}

// GetIDsFromTaskContext returns IDs obtained from the context passed into tasks.
func GetIDsFromTaskContext(ctx context.Context) TaskContextIDs {
	iidAny := ctx.Value("iid")
	tidAny := ctx.Value("tid")
	ridAny := ctx.Value("rid")
	iid := ""
	tid := taskid.TaskImplementationId{}
	rid := ""
	if iidAny != nil {
		iid = iidAny.(string)
	}
	if tidAny != nil {
		tid = tidAny.(taskid.TaskImplementationId)
	}
	if ridAny != nil {
		rid = ridAny.(string)
	}
	return TaskContextIDs{
		InspectionID: iid,
		TaskID:       tid,
		RunID:        rid,
	}

}
