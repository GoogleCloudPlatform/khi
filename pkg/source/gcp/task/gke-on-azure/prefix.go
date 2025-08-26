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

package azure

import (
	"context"

	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
)

var AnthosOnAzureClusterNamePrefixTask = common_task.NewTask(taskid.NewImplementationID(task.ClusterNamePrefixTaskID, "gke-on-azure"), []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "azureClusters/", nil
}, inspection_contract.InspectionTypeLabel(InspectionTypeId))
