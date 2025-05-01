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

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	gke_audit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/gke_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateGKEAuditQuery(projectName string, clusterName string) string {
	return fmt.Sprintf(`resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/%s/logs/cloudaudit.googleapis.com%%2Factivity"
resource.labels.cluster_name="%s"`, projectName, clusterName)
}

var GKEAuditQueryTask = query.NewQueryGeneratorTask(gke_audit_taskid.GKEAuditLogQueryTaskID, "GKE Audit logs", enum.LogTypeGkeAudit, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID.Ref(),
	gcp_task.InputClusterNameTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())

	return []string{GenerateGKEAuditQuery(projectID, clusterName)}, nil
}, GenerateGKEAuditQuery("gcp-project-id", "gcp-cluster-name"))
