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

package googlecloudclustergkeonaws_impl

import (
	"context"

	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergkeonaws_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AnthosOnAWSClusterNamePrefixTask is a task that provides the cluster name prefix for GKE on AWS.
var AnthosOnAWSClusterNamePrefixTask = common_task.NewTask(googlecloudclustergkeonaws_contract.ClusterNamePrefixTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "awsClusters/", nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustergkeonaws_contract.InspectionTypeId))
