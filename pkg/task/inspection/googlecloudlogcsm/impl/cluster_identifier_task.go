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

package googlecloudlogcsm_impl

import (
	"context"
	"strings"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
)

// CSMClusterIdentifierTask extracts the unique cluster identifier(s) from BackendService names.
// CSM BackendService names follow the pattern: gsmrsvd-(cluster-identifier)-(neg-id).
var CSMClusterIdentifierTask = coretask.NewTask(
	googlecloudlogcsm_contract.CSMClusterIdentifierTaskID,
	[]taskid.UntypedTaskReference{googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref()},
	func(ctx context.Context) ([]string, error) {
		inventory := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref())

		uniqueIds := make(map[string]struct{})

		for _, bsName := range inventory {
			if strings.HasPrefix(bsName, "gsmrsvd-") {
				// gsmrsvd-<cluster-id>-<neg-id>
				parts := strings.Split(bsName, "-")
				if len(parts) >= 3 {
					uniqueIds[parts[1]] = struct{}{}
				}
			}
		}

		result := make([]string, 0, len(uniqueIds))
		for id := range uniqueIds {
			result = append(result, id)
		}
		return result, nil
	},
)
