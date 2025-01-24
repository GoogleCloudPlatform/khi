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
	"fmt"
	"log/slog"

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var AutocompleteClusterNames = task.NewCachedProcessor(gcp_task.AutocompleteClusterNamesTaskID+"#anthos-on-vmware", []string{
	gcp_task.InputProjectIdTaskID,
}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return nil, err
	}
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	if projectId != "" {
		clusterNames, err := client.GetAnthosOnVMWareClusterNames(ctx, projectId)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the cluster names in the project %s\n%s", projectId, err))
			return &gcp_task.AutocompleteClusterNameList{
				ClusterNames: []string{},
				Error:        "Failed to get the list from API",
			}, nil
		}
		return &gcp_task.AutocompleteClusterNameList{
			ClusterNames: clusterNames,
			Error:        "",
		}, nil
	}
	return &gcp_task.AutocompleteClusterNameList{
		ClusterNames: []string{},
		Error:        "Project ID is empty",
	}, nil
}, inspection_task.InspectionTypeLabel(InspectionTypeId))
