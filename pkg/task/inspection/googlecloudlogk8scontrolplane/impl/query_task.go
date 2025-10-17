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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func GenerateK8sControlPlaneQuery(clusterName string, projectId string, controlplaneComponentFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_control_plane_component"
resource.labels.cluster_name="%s"
resource.labels.project_id="%s"
-sourceLocation.file="httplog.go"
%s`, clusterName, projectId, generateK8sControlPlaneComponentFilter(controlplaneComponentFilter))
}

var GKEK8sControlPlaneLogQueryTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudlogk8scontrolplane_contract.GKEK8sControlPlaneComponentQueryTaskID, "K8s control plane logs", enum.LogTypeControlPlaneComponent, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	projectId := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	controlplaneComponentNameFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref())

	return []string{GenerateK8sControlPlaneQuery(clusterName, projectId, controlplaneComponentNameFilter)}, nil
}, GenerateK8sControlPlaneQuery("gcp-cluster-name", "gcp-project-id", &gcpqueryutil.SetFilterParseResult{
	SubtractMode: true,
}))

func generateK8sControlPlaneComponentFilter(filter *gcpqueryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate component name filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No component name filter"
		}
		return fmt.Sprintf(`-resource.labels.component_name:(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(filter.Subtractives), " OR "))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`
		}
		return fmt.Sprintf(`resource.labels.component_name:(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(filter.Additives), " OR "))
	}
}
