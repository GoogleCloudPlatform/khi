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

package gke_audit

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func GenerateGKEAuditQuery(projectName string, clusterName string) string {
	return fmt.Sprintf(`resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/%s/logs/cloudaudit.googleapis.com%%2Factivity"
resource.labels.cluster_name="%s"`, projectName, clusterName)
}

var GKEAuditLogQueryTaskID = query.GKEQueryPrefix + "gke-audit"
var GKEAuditQueryTask = query.NewQueryGeneratorTask(GKEAuditLogQueryTaskID, "GKE Audit logs", enum.LogTypeGkeAudit, []string{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}

	return []string{GenerateGKEAuditQuery(projectId, clusterName)}, nil
}, GenerateGKEAuditQuery("gcp-project-id", "gcp-cluster-name"))
