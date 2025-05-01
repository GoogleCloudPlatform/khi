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

package k8scontrolplanecomponent

import (
	"context"
	"fmt"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	k8s_control_plane_component_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_control_plane_component/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateK8sControlPlaneQuery(clusterName string, projectId string, controlplaneComponentFilter *queryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_control_plane_component"
resource.labels.cluster_name="%s"
resource.labels.project_id="%s"
-sourceLocation.file="httplog.go"
%s`, clusterName, projectId, generateK8sControlPlaneComponentFilter(controlplaneComponentFilter))
}

var GKEK8sControlPlaneLogQueryTask = query.NewQueryGeneratorTask(k8s_control_plane_component_taskid.GKEK8sControlPlaneComponentQueryTaskID, "K8s control plane logs", enum.LogTypeControlPlaneComponent, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID.Ref(),
	gcp_task.InputClusterNameTaskID.Ref(),
	k8s_control_plane_component_taskid.InputControlPlaneComponentNameFilterTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())
	projectId := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	controlplaneComponentNameFilter := task.GetTaskResult(ctx, k8s_control_plane_component_taskid.InputControlPlaneComponentNameFilterTaskID.Ref())

	return []string{GenerateK8sControlPlaneQuery(clusterName, projectId, controlplaneComponentNameFilter)}, nil
}, GenerateK8sControlPlaneQuery("gcp-cluster-name", "gcp-project-id", &queryutil.SetFilterParseResult{
	SubtractMode: true,
}))

func generateK8sControlPlaneComponentFilter(filter *queryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate component name filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No component name filter"
		}
		return fmt.Sprintf(`-resource.labels.component_name:(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(filter.Subtractives), " OR "))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`
		}
		return fmt.Sprintf(`resource.labels.component_name:(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(filter.Additives), " OR "))
	}
}
