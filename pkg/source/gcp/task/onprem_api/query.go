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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var OnPremCloudAPIQueryTaskID = query.GKEQueryPrefix + "onprem-api"

func GenerateOnPremAPIQuery(clusterNameWithPrefix string) string {
	return fmt.Sprintf(`resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"%s"
`, clusterNameWithPrefix)
}

var OnPremAPIQueryTask = query.NewQueryGeneratorTask(OnPremCloudAPIQueryTaskID, "OnPrem API Logs", enum.LogTypeOnPremAPI, []string{
	gcp_task.InputClusterNameTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return []string{GenerateOnPremAPIQuery(clusterName)}, nil
})
