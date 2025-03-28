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

package baremetal

import (
	"context"
	"fmt"
	"log/slog"

	inspection_cached_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/cached_task"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var AutocompleteClusterNames = inspection_cached_task.NewCachedTask(taskid.NewImplementationID(gcp_task.AutocompleteClusterNamesTaskID, "anthos-on-baremetal"), []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID,
}, func(ctx context.Context, prevValue inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]) (inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList], error) {
	client, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{}, err
	}
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())

	if projectID == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID != "" {
		clusterNames, err := client.GetAnthosOnBaremetalClusterNames(ctx, projectID)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to read the cluster names in the project %s\n%s", projectID, err))
			return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
				DependencyDigest: projectID,
				Value: &gcp_task.AutocompleteClusterNameList{
					ClusterNames: []string{},
					Error:        "Failed to get the list from API",
				},
			}, nil
		}
		return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
			DependencyDigest: projectID,
			Value: &gcp_task.AutocompleteClusterNameList{
				ClusterNames: clusterNames,
				Error:        "",
			},
		}, nil
	}
	return inspection_cached_task.PreviousTaskResult[*gcp_task.AutocompleteClusterNameList]{
		DependencyDigest: projectID,
		Value: &gcp_task.AutocompleteClusterNameList{
			ClusterNames: []string{},
			Error:        "Project ID is empty",
		},
	}, nil
}, inspection_task.InspectionTypeLabel(InspectionTypeId))
