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

package onprem_api

import (
	"context"
	"fmt"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	onprem_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/onprem_api/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
)

func GenerateOnPremAPIQuery(clusterNameWithPrefix string) string {
	return fmt.Sprintf(`resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"%s"
`, clusterNameWithPrefix)
}

var OnPremAPIQueryTask = query.NewQueryGeneratorTask(onprem_api_taskid.OnPremCloudAPIQueryTaskID, "OnPrem API Logs", enum.LogTypeOnPremAPI, []taskid.UntypedTaskReference{
	gcp_task.InputClusterNameTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())
	return []string{GenerateOnPremAPIQuery(clusterName)}, nil
}, GenerateOnPremAPIQuery("baremetalClusters/my-cluster"))
