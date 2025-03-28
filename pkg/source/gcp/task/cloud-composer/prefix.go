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

package composer_task

import (
	"context"

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var ComposerClusterNamePrefixTask = common_task.NewTask(taskid.NewImplementationID(task.ClusterNamePrefixTaskID, "composer"), []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "", nil
}, inspection_task.InspectionTypeLabel(composer_inspection_type.InspectionTypeId))
