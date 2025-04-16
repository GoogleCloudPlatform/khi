// Copyright 2024 Google LLC
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

package inspection_task

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var BuilderGeneratorTaskID = taskid.NewDefaultImplementationID[*history.Builder](InspectionTaskPrefix + "builder-generator")

var BuilderGeneratorTask = task.NewTask(BuilderGeneratorTaskID, []taskid.UntypedTaskReference{ioconfig.IOConfigTaskID}, func(ctx context.Context) (*history.Builder, error) {
	ioConfig := task.GetTaskResult(ctx, ioconfig.IOConfigTaskID.GetTaskReference())
	return history.NewBuilder(ioConfig), nil
})
