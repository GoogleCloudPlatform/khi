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

package aws

import (
	"context"

	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

var AnthosOnAWSClusterNamePrefixTask = common_task.NewTask(taskid.NewImplementationID(googlecloudk8scommon_contract.ClusterNamePrefixTaskID, "gke-on-aws"), []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "awsClusters/", nil
}, inspection_contract.InspectionTypeLabel(InspectionTypeId))
