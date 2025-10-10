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

package googlecloudloggkeautoscaler_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func generateAutoscalerQuery(projectId string, clusterName string, excludeStatus bool) string {
	excludeStatusQueryFragment := "-- include query for status log"
	if excludeStatus {
		excludeStatusQueryFragment = `-jsonPayload.status: ""`
	}
	return fmt.Sprintf(`resource.type="k8s_cluster"
resource.labels.project_id="%s"
resource.labels.cluster_name="%s"
%s
logName="projects/%s/logs/container.googleapis.com%%2Fcluster-autoscaler-visibility"`, projectId, clusterName, excludeStatusQueryFragment, projectId)
}

var AutoscalerQueryTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudloggkeautoscaler_contract.AutoscalerQueryTaskID, "Autoscaler logs", enum.LogTypeAutoscaler, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())

	return []string{generateAutoscalerQuery(projectID, clusterName, true)}, nil
}, generateAutoscalerQuery("gcp-project-id", "gcp-cluster-name", true))
