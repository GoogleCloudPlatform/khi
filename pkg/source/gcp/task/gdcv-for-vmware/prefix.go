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

package vmware

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
)

var AnthosOnVMWareClusterNamePrefixTask = inspection_task.NewInspectionProducer(task.ClusterNamePrefixTaskID+"#gdcv-for-vmware", func(ctx context.Context, taskMode int, progress *progress.TaskProgress) (any, error) {
	return "", nil
}, inspection_task.InspectionTypeLabel(InspectionTypeId))
