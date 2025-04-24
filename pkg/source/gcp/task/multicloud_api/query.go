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

package multicloud_api

import (
	"context"
	"fmt"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	multicloud_api_taskidvar "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/multicloud_api/multicloud_api_taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
)

func GenerateMultiCloudAPIQuery(clusterNameWithPrefix string) string {
	return fmt.Sprintf(`resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"%s"
`, clusterNameWithPrefix)
}

var MultiCloudAPIQueryTask = query.NewQueryGeneratorTask(multicloud_api_taskidvar.MultiCloudAPIQueryTaskID, "Multicloud API Logs", enum.LogTypeMulticloudAPI, []taskid.UntypedTaskReference{
	gcp_task.InputClusterNameTaskID,
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.GetTaskReference())

	return []string{GenerateMultiCloudAPIQuery(clusterName)}, nil
}, GenerateMultiCloudAPIQuery("awsClusters/cluster-foo"))
